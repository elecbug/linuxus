package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/elecbug/linuxus/src/manager/internal/packet"
)

// HandleUserSessionState records active session counts reported by auth service.
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

	var req packet.SessionStateReport
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

// updateSessionState updates runtime idle/session tracking state for a user.
func (s *Server) updateSessionState(userID string, active int, observedAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rt, ok := s.runtimes[userID]
	if !ok {
		rt = &RuntimeState{
			UserID: userID,
		}
		s.runtimes[userID] = rt
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

// stopAndRemoveUserContainerAndNetwork stops and removes user runtime resources.
func (s *Server) stopAndRemoveUserContainerAndNetwork(ctx context.Context, userID string) error {
	containerName, networkName := s.resolveUserRuntimeNames(ctx, userID)
	stopCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	timeoutSec := 10
	err := s.docker.ContainerStop(stopCtx, containerName, container.StopOptions{
		Timeout: &timeoutSec,
	})
	if err != nil {
		log.Printf("container stop warning for %s: %v", userID, err)
	}

	removeCtx, removeCancel := context.WithTimeout(ctx, 15*time.Second)
	defer removeCancel()

	if err := s.docker.ContainerRemove(removeCtx, containerName, container.RemoveOptions{
		Force:         true,
		RemoveVolumes: false,
	}); err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("container remove failed: %w", err)
		}
		log.Printf("container already removed for %s, continuing cleanup", userID)
	}

	if err := s.disconnectAuthFromUserNetwork(ctx, networkName); err != nil {
		log.Printf("auth disconnect warning for %s: %v", userID, err)
	}

	if err := s.removeNetwork(ctx, networkName); err != nil {
		return fmt.Errorf("network remove failed: %w", err)
	}

	return nil
}

// resolveUserRuntimeNames resolves managed container/network names for a user.
func (s *Server) resolveUserRuntimeNames(ctx context.Context, userID string) (string, string) {
	if !allowID(userID) {
		return "", ""
	}

	containerName := s.cfg.UserContainerNamePrefix + userID
	networkName := s.cfg.NetworkPrefix + userID

	inspect, err := s.docker.ContainerInspect(ctx, containerName)
	if err != nil {
		return containerName, networkName
	}

	if inspect.NetworkSettings != nil {
		for attachedNetworkName := range inspect.NetworkSettings.Networks {
			if strings.HasPrefix(attachedNetworkName, s.cfg.NetworkPrefix) {
				networkName = attachedNetworkName
				break
			}
		}
	}

	return containerName, networkName
}

// disconnectAuthFromUserNetwork detaches the auth container from a user network.
func (s *Server) disconnectAuthFromUserNetwork(ctx context.Context, networkName string) error {
	if err := s.docker.NetworkDisconnect(ctx, networkName, s.cfg.AuthContainerName, true); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") ||
			strings.Contains(strings.ToLower(err.Error()), "already disconnected") {
			return nil
		}
		return fmt.Errorf("failed to disconnect auth container from %s: %w", networkName, err)
	}

	return nil
}

// removeNetwork removes a user network if present.
func (s *Server) removeNetwork(ctx context.Context, name string) error {
	if err := s.docker.NetworkRemove(ctx, name); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return nil
		}
		return fmt.Errorf("network remove failed: %w", err)
	}
	return nil
}
