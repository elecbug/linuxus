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

// reInvalid matches characters that are not allowed in sanitized runtime names.
var reInvalid = regexp.MustCompile(`[^a-z0-9]+`)

// Server holds manager service dependencies and runtime state tracking.
type Server struct {
	// docker is the Docker API client.
	docker *client.Client
	// mux is the HTTP route multiplexer.
	mux *http.ServeMux
	// cfg is the active runtime configuration.
	cfg *config.Config

	// mu protects runtimes map access.
	mu sync.Mutex
	// runtimes tracks active user runtimes by sanitized user ID.
	runtimes map[string]*RuntimeState
}

// RuntimeState tracks observed session activity for one user runtime.
type RuntimeState struct {
	// UserID is the original user identifier.
	UserID string
	// ActiveSessions is the current number of active sessions.
	ActiveSessions int
	// LastObservedAt is the most recent session state observation time.
	LastObservedAt time.Time
	// IdleSince is when active sessions dropped to zero.
	IdleSince time.Time
}

// NewServer creates a manager server and initializes the Docker client.
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

// RegisterRoutes registers all HTTP endpoints served by manager.
func (s *Server) RegisterRoutes() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.HandleHealthz)
	mux.HandleFunc("/user/up", s.HandleUserUp)
	mux.HandleFunc("/user/session-state", s.HandleUserSessionState)

	s.mux = mux
}

// Start runs the HTTP server and handles graceful shutdown on signals.
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

// Close releases server resources.
func (s *Server) Close() error {
	if s == nil || s.docker == nil {
		return nil
	}

	return s.docker.Close()
}

// StartIdleReaper starts periodic cleanup for idle user runtimes.
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

// reapIdleContainers removes user runtimes that have been idle past timeout.
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

// HandleHealthz responds with a simple readiness payload.
func (s *Server) HandleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true,
	})
}
