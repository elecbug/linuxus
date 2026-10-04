package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/docker/docker/client"
	"github.com/elecbug/linuxus/src/internal/common/diskservice"
	"github.com/elecbug/linuxus/src/internal/common/http_helper"
	"github.com/elecbug/linuxus/src/internal/manager/config"
)

// Server holds manager service dependencies and runtime state tracking.
type Server struct {
	// docker is the Docker API client.
	docker *client.Client
	// mux is the HTTP route multiplexer.
	mux *http.ServeMux
	// cfg is the active runtime configuration.
	cfg *config.Config

	// prepareMu serializes disk preparation, runtime allocation and idle cleanup.
	pending    atomic.Int32
	prepareMu  runtimeLock
	diskClient *http.Client

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
	// LastPreparedAt protects a newly prepared runtime until session reporting starts.
	LastPreparedAt time.Time
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
		docker:     cli,
		diskClient: diskservice.NewClient(cfg.DiskServiceSocket, cfg.ManagerWaitTime),
		cfg:        cfg,

		mu:       sync.Mutex{},
		runtimes: make(map[string]*RuntimeState),
	}, nil
}

// RegisterRoutes registers all HTTP endpoints served by manager.
func (s *Server) RegisterRoutes() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.HandleHealthz)
	mux.HandleFunc("/admin/user", s.HandleAdminUser)
	mux.HandleFunc("/admin/status", s.HandleAdminStatus)
	mux.HandleFunc("/user/up", s.HandleUserUp)
	mux.HandleFunc("/user/session-state", s.HandleUserSessionState)
	mux.HandleFunc("/user/session-snapshot", s.HandleSessionSnapshot)

	s.mux = mux
}

// Start runs the HTTP server and handles graceful shutdown on signals.
func (s *Server) Start() {
	ctx, cancel := context.WithCancel(context.Background())

	reconcileCtx, stopReconcile := context.WithTimeout(ctx, 10*time.Second)
	if err := s.reconcileRuntimes(reconcileCtx); err != nil {
		log.Printf("runtime reconciliation: %v", err)
	}
	stopReconcile()
	s.StartIdleReaper(ctx)
	s.startReconciliation(ctx)

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

	if s.diskClient != nil {
		s.diskClient.CloseIdleConnections()
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
		if err := s.reapIdleUser(ctx, userID); err != nil {
			log.Printf("idle cleanup failed for %s: %v", userID, err)
		}
	}
}

func (s *Server) reapIdleUser(ctx context.Context, userID string) error {
	if err := s.prepareMu.LockContext(ctx); err != nil {
		return err
	}
	defer s.prepareMu.Unlock()

	// A user-up request may have refreshed the idle deadline after selection.
	s.mu.Lock()
	rt, ok := s.runtimes[userID]
	if !ok || rt == nil || rt.ActiveSessions > 0 || rt.IdleSince.IsZero() ||
		time.Since(rt.IdleSince) <= s.cfg.ContainerTimeout {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	if err := s.stopAndRemoveUserContainerAndNetwork(ctx, userID); err != nil {
		return err
	}

	s.mu.Lock()
	// Preserve a session update that arrived during the Docker calls.
	if current := s.runtimes[userID]; current == rt && current.ActiveSessions == 0 &&
		!current.IdleSince.IsZero() && time.Since(current.IdleSince) > s.cfg.ContainerTimeout {
		delete(s.runtimes, userID)
	}
	s.mu.Unlock()
	log.Printf("idle container cleaned up for %s", userID)
	return nil
}

// markRuntimeReady grants time for Auth to establish and report the WebSocket.
func (s *Server) markRuntimeReady(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runtimes == nil {
		s.runtimes = make(map[string]*RuntimeState)
	}
	rt := s.runtimes[userID]
	if rt == nil {
		rt = &RuntimeState{UserID: userID}
		s.runtimes[userID] = rt
	}
	rt.LastPreparedAt = time.Now()
	if rt.ActiveSessions == 0 {
		rt.IdleSince = rt.LastPreparedAt
	}
}

// HandleHealthz responds with a simple readiness payload.
func (s *Server) HandleHealthz(w http.ResponseWriter, r *http.Request) {
	http_helper.WriteJSONViaHTTP(w, http.StatusOK, map[string]any{
		"ok": true,
	})
}
