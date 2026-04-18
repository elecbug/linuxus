package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func (s *Server) HandleUserSessionState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.cfg.ManagerSecret != "" {
		if r.Header.Get("X-Manager-Secret") != s.cfg.ManagerSecret {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var req sessionStateReport
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		http.Error(w, "missing user_id", http.StatusBadRequest)
		return
	}

	if req.ActiveSessions < 0 {
		http.Error(w, "invalid active_sessions", http.StatusBadRequest)
		return
	}

	if req.ObservedAt.IsZero() {
		req.ObservedAt = time.Now()
	}

	s.updateSessionState(req.UserID, req.ActiveSessions, req.ObservedAt)

	w.WriteHeader(http.StatusOK)
}

func (s *Server) updateSessionState(userID string, active int, observedAt time.Time) {
	safeID := sanitizeID(userID)

	s.mu.Lock()
	defer s.mu.Unlock()

	rt, ok := s.runtimes[safeID]
	if !ok {
		rt = &RuntimeState{
			UserID: userID,
		}
		s.runtimes[safeID] = rt
	}

	prev := rt.ActiveSessions

	rt.ActiveSessions = active
	rt.LastObservedAt = observedAt

	if active > 0 {
		rt.IdleSince = time.Time{}
	} else {
		if prev > 0 {
			rt.IdleSince = observedAt
		} else if rt.IdleSince.IsZero() {
			rt.IdleSince = observedAt
		}
	}
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
