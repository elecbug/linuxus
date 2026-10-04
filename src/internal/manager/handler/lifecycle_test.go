package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/elecbug/linuxus/src/internal/manager/config"
)

func TestReconnectResetsIdleDeadline(t *testing.T) {
	d := &runtimeDocker{exists: true, running: true}
	s := runtimeTestServer(t, d, nil)
	s.cfg.ContainerTimeout = time.Minute
	s.runtimes = map[string]*RuntimeState{"alice": {UserID: "alice", IdleSince: time.Now().Add(-time.Hour)}}
	if _, err := s.ensureUserRuntimeReady(context.Background(), "alice"); err != nil {
		t.Fatal(err)
	}
	s.reapIdleContainers(context.Background())
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.mutations) != 0 {
		t.Fatalf("reaper stopped a reconnecting user: %v", d.mutations)
	}
}

func TestUserUpDeadlineWhileQueued(t *testing.T) {
	s := &Server{}
	s.prepareMu.Lock()
	defer s.prepareMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := s.ensureUserRuntimeReady(ctx, "alice"); done <- err }()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal(err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("queued request ignored its deadline")
	}
}

func TestSessionStateIgnoresOutOfOrderReports(t *testing.T) {
	s := &Server{runtimes: make(map[string]*RuntimeState)}
	now := time.Now()
	s.updateSessionState("alice", 1, now)
	s.updateSessionState("alice", 0, now.Add(-time.Hour))
	if rt := s.runtimes["alice"]; rt.ActiveSessions != 1 || !rt.IdleSince.IsZero() {
		t.Fatalf("old report reset a live session: %+v", rt)
	}
}

func TestSessionStateRejectsInvalidUser(t *testing.T) {
	s := &Server{cfg: &config.Config{}, runtimes: make(map[string]*RuntimeState)}
	for _, id := range []string{"../outside", "a/b", " ", ""} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/user/session-state", strings.NewReader(`{"user_id":"`+id+`","active_sessions":0}`))
		s.HandleUserSessionState(w, r)
		if w.Code != http.StatusBadRequest {
			t.Errorf("accepted invalid user %q: %d", id, w.Code)
		}
	}
}

func TestLateIdleReportPreservesReconnectGrace(t *testing.T) {
	s := &Server{runtimes: make(map[string]*RuntimeState)}
	s.markRuntimeReady("alice")
	prepared := s.runtimes["alice"].IdleSince
	s.updateSessionState("alice", 0, prepared.Add(-time.Hour))
	if s.runtimes["alice"].IdleSince.Before(prepared) {
		t.Fatal("late report expired the reconnect grace period")
	}
}

func TestReaperWaitsForPreparationAndRechecksState(t *testing.T) {
	d := &runtimeDocker{exists: true, running: true}
	s := runtimeTestServer(t, d, nil)
	s.cfg.ContainerTimeout = time.Minute
	s.runtimes = map[string]*RuntimeState{"alice": {UserID: "alice", IdleSince: time.Now().Add(-time.Hour)}}
	s.prepareMu.Lock()
	var once sync.Once
	unlock := func() { once.Do(s.prepareMu.Unlock) }
	defer unlock()
	done := make(chan error, 1)
	go func() { done <- s.reapIdleUser(context.Background(), "alice") }()
	select {
	case err := <-done:
		t.Fatalf("reaper ran during runtime preparation: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	s.markRuntimeReady("alice")
	unlock()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("reaper did not resume")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.mutations) != 0 {
		t.Fatalf("reaper ignored the updated deadline: %v", d.mutations)
	}
}
