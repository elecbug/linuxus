package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/client"
	"github.com/elecbug/linuxus/src/manager/internal/config"
)

var reInvalid = regexp.MustCompile(`[^a-z0-9]+`)

type Server struct {
	docker *client.Client
	mux    *http.ServeMux
	// cfg is the active runtime configuration.
	cfg *config.Config

	// mu protects runtimes map access.
	mu sync.Mutex
	// runtimes tracks active user runtimes by sanitized user ID.
	runtimes map[string]*RuntimeState
}

type RuntimeState struct {
	UserID         string
	ActiveSessions int
	LastObservedAt time.Time
	IdleSince      time.Time
}

func NewServer(cfg *config.Config) (*Server, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker client init failed: %w", err)
	}

	return &Server{
		docker: cli,
		cfg:    cfg,

		mu:       sync.Mutex{},
		runtimes: make(map[string]*RuntimeState),
	}, nil
}

func (s *Server) RegisterRoutes() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.HandleHealthz)
	mux.HandleFunc("/user/up", s.HandleUserUp)
	mux.HandleFunc("/user/session-state", s.HandleUserSessionState)

	s.mux = mux
}

func (s *Server) Start() {
	ctx, cancel := context.WithCancel(context.Background())

	s.StartIdleReaper(ctx)

	srv := &http.Server{
		Addr:              s.cfg.ListenAddr,
		Handler:           s.mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("manager listening on %s", s.cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen failed: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh

	cancel()

	ctx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(ctx)
	_ = srv.Close()
}

func (s *Server) Close() error {
	if s == nil || s.docker == nil {
		return nil
	}

	return s.docker.Close()
}

func (s *Server) StartIdleReaper(ctx context.Context) {
	timeout := s.cfg.ContainerTimeout
	if timeout <= 0 {
		return
	}

	interval := time.Minute
	if timeout < interval {
		interval = timeout / 2
		if interval < 5*time.Second {
			interval = 5 * time.Second
		}
	}

	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.reapIdleContainers(ctx)
			}
		}
	}()
}

func (s *Server) reapIdleContainers(ctx context.Context) {
	var candidates []string
	now := time.Now()

	s.mu.Lock()
	for userID, rt := range s.runtimes {
		if rt == nil {
			continue
		}
		if rt.ActiveSessions > 0 {
			continue
		}
		if rt.IdleSince.IsZero() {
			continue
		}
		if now.Sub(rt.IdleSince) > s.cfg.ContainerTimeout {
			candidates = append(candidates, userID)
		}
	}
	s.mu.Unlock()

	for _, userID := range candidates {
		s.mu.Lock()
		rt, ok := s.runtimes[userID]
		if !ok || rt == nil || rt.ActiveSessions > 0 || rt.IdleSince.IsZero() ||
			time.Since(rt.IdleSince) <= s.cfg.ContainerTimeout {
			s.mu.Unlock()
			continue
		}
		s.mu.Unlock()

		if err := s.stopAndRemoveUserContainerAndNetwork(ctx, userID); err != nil {
			log.Printf("idle cleanup failed for %s: %v", userID, err)
			continue
		}

		s.mu.Lock()
		delete(s.runtimes, userID)
		s.mu.Unlock()

		log.Printf("idle container cleaned up for %s", userID)
	}
}

func (s *Server) HandleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true,
	})
}
