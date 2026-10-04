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
	s.mu.Lock()
	ids := make([]string, 0, len(s.runtimes))
	for id := range s.runtimes {
		ids = append(ids, id)
	}
	s.mu.Unlock()
	for id, count := range snapshot.Sessions {
		s.updateSessionState(id, count, snapshot.ObservedAt)
	}
	for _, id := range ids {
		if _, ok := snapshot.Sessions[id]; !ok {
			s.updateSessionState(id, 0, snapshot.ObservedAt)
		}
	}
	w.WriteHeader(200)
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
