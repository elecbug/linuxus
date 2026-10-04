package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/elecbug/linuxus/src/internal/common/packet"
	"github.com/elecbug/linuxus/src/internal/common/ruleset"
)

func (s *Server) HandleSessionSnapshot(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeAdmin(w, r, http.MethodPost) {
		return
	}
	var snapshot packet.SessionSnapshot
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&snapshot); err != nil || snapshot.ObservedAt.IsZero() || snapshot.ObservedAt.After(time.Now().Add(time.Minute)) {
		http.Error(w, "invalid session snapshot", 400)
		return
	}
	for id, count := range snapshot.Sessions {
		if !ruleset.AllowedUserID(id) || count < 0 {
			http.Error(w, "invalid session state", 400)
			return
		}
	}
	s.applySessionSnapshot(snapshot)
	w.WriteHeader(200)
}

// Apply the complete observation atomically, including the absence of users.
func (s *Server) applySessionSnapshot(snapshot packet.SessionSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !snapshot.ObservedAt.After(s.lastSnapshotAt) {
		return
	}
	s.lastSnapshotAt = snapshot.ObservedAt
	for id, count := range snapshot.Sessions {
		s.updateSessionStateLocked(id, count, snapshot.ObservedAt)
	}
	for id := range s.runtimes {
		if _, ok := snapshot.Sessions[id]; !ok {
			s.updateSessionStateLocked(id, 0, snapshot.ObservedAt)
		}
	}
}

func (s *Server) startReconciliation(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				if err := s.reconcileRuntimes(checkCtx); err != nil {
					log.Printf("runtime reconciliation: %v", err)
				}
				cancel()
				// No positive count may survive forever after Auth disappears.
				s.mu.Lock()
				for _, state := range s.runtimes {
					if state != nil && state.ActiveSessions > 0 && time.Since(state.LastObservedAt) > 2*time.Minute {
						state.ActiveSessions = 0
						state.IdleSince = time.Now()
					}
				}
				s.mu.Unlock()
			}
		}
	}()
}
