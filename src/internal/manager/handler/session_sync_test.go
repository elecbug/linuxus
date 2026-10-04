package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/elecbug/linuxus/src/internal/common/packet"
	"github.com/elecbug/linuxus/src/internal/manager/config"
)

func TestSnapshotRejectsDelayedReportsForUnknownAndRemovedUsers(t *testing.T) {
	now := time.Now()
	s := &Server{}
	s.applySessionSnapshot(packet.SessionSnapshot{ObservedAt: now, Sessions: map[string]int{}})
	for _, at := range []time.Time{now.Add(-time.Second), now} {
		s.updateSessionState("alice", 1, at)
		if len(s.runtimes) != 0 {
			t.Fatal("delayed report resurrected an absent session")
		}
	}
	s.updateSessionState("alice", 1, now.Add(time.Second))
	if s.runtimes["alice"].ActiveSessions != 1 {
		t.Fatal("new session report was ignored")
	}
	s.applySessionSnapshot(packet.SessionSnapshot{ObservedAt: now.Add(2 * time.Second), Sessions: map[string]int{}})
	delete(s.runtimes, "alice") // Model completed idle cleanup.
	s.applySessionSnapshot(packet.SessionSnapshot{ObservedAt: now, Sessions: map[string]int{"alice": 1}})
	if len(s.runtimes) != 0 {
		t.Fatal("older full snapshot recreated a cleaned runtime")
	}
}

func TestConcurrentSnapshotsKeepNewestObservation(t *testing.T) {
	s := &Server{}
	now := time.Now()
	var workers sync.WaitGroup
	for i := 0; i < 100; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			s.applySessionSnapshot(packet.SessionSnapshot{ObservedAt: now.Add(time.Duration(i) * time.Millisecond), Sessions: map[string]int{"alice": i, "bob": i}})
		}(i)
	}
	workers.Wait()
	for _, id := range []string{"alice", "bob"} {
		if s.runtimes[id].ActiveSessions != 99 {
			t.Fatalf("%s retained a stale snapshot: %+v", id, s.runtimes[id])
		}
	}
	s.updateSessionState("alice", 101, now.Add(time.Second))
	s.applySessionSnapshot(packet.SessionSnapshot{ObservedAt: now.Add(500 * time.Millisecond), Sessions: map[string]int{}})
	if s.runtimes["alice"].ActiveSessions != 101 || s.runtimes["bob"].ActiveSessions != 0 {
		t.Fatal("snapshot overwrote a newer individual report or missed an absent user")
	}
}

func TestSessionReportsRejectFutureTimestamps(t *testing.T) {
	s := &Server{cfg: &config.Config{ManagerSessionSecret: "secret"}}
	body, err := json.Marshal(packet.SessionStateReport{UserID: "alice", ActiveSessions: 1, ObservedAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "/user/session-state", strings.NewReader(string(body)))
	r.Header.Set("X-Manager-Session-Secret", "secret")
	w := httptest.NewRecorder()
	s.HandleUserSessionState(w, r)
	if w.Code != http.StatusBadRequest || len(s.runtimes) != 0 {
		t.Fatal("future report could suppress later session updates")
	}
}
