package handler

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/elecbug/linuxus/src/internal/common/packet"
	"github.com/elecbug/linuxus/src/internal/common/ruleset"
)

const userContainerNetworkLabel = "io.linuxus.user-container"

// HandleUserSessionState records active session counts reported by auth service.
func (s *Server) HandleUserSessionState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.cfg.ManagerSessionSecret != "" {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Manager-Session-Secret")), []byte(s.cfg.ManagerSessionSecret)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var req packet.SessionStateReport
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	if !ruleset.AllowedUserID(req.UserID) {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	if req.ActiveSessions < 0 {
		http.Error(w, "invalid active_sessions", http.StatusBadRequest)
		return
	}

	if req.ObservedAt.IsZero() {
		req.ObservedAt = time.Now()
	}

	if req.ObservedAt.After(time.Now().Add(time.Minute)) {
		http.Error(w, "invalid observed_at", http.StatusBadRequest)
		return
	}

	s.updateSessionState(req.UserID, req.ActiveSessions, req.ObservedAt)

	w.WriteHeader(http.StatusOK)
}

// updateSessionState updates runtime idle/session tracking state for a user.
func (s *Server) updateSessionState(userID string, active int, observedAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.lastSnapshotAt.IsZero() && !observedAt.After(s.lastSnapshotAt) {
		return
	}
	s.updateSessionStateLocked(userID, active, observedAt)
}

// The caller holds mu for the whole update or complete snapshot.
func (s *Server) updateSessionStateLocked(userID string, active int, observedAt time.Time) {
	if s.runtimes == nil {
		s.runtimes = make(map[string]*RuntimeState)
	}
	rt := s.runtimes[userID]
	if rt == nil {
		rt = &RuntimeState{
			UserID: userID,
		}
		s.runtimes[userID] = rt
	}

	if observedAt.Before(rt.LastObservedAt) {
		return
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
		if rt.IdleSince.Before(rt.LastPreparedAt) {
			rt.IdleSince = rt.LastPreparedAt
		}
	}
}

// stopAndRemoveUserContainerAndNetwork stops and removes user runtime resources.
func (s *Server) stopAndRemoveUserContainerAndNetwork(ctx context.Context, userID string) error {
	containerID, networkIDs, err := s.resolveUserRuntimeResources(ctx, userID)
	if err != nil {
		return err
	}
	if containerID != "" {
		stopCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		timeoutSec := 10
		if err := s.docker.ContainerStop(stopCtx, containerID, container.StopOptions{Timeout: &timeoutSec}); err != nil {
			log.Printf("container stop warning for %s: %v", userID, err)
		}

		removeCtx, removeCancel := context.WithTimeout(ctx, 15*time.Second)
		defer removeCancel()

		if err := s.docker.ContainerRemove(removeCtx, containerID, container.RemoveOptions{Force: true}); err != nil {
			if !errdefs.IsNotFound(err) {
				return fmt.Errorf("container remove failed: %w", err)
			}
			log.Printf("container already removed for %s, continuing cleanup", userID)
		}
	}

	var failures []error
	for _, networkID := range networkIDs {
		if err := s.disconnectAuthFromUserNetwork(ctx, networkID); err != nil {
			log.Printf("auth disconnect warning for %s: %v", userID, err)
		}
		if err := s.removeNetwork(ctx, networkID); err != nil {
			failures = append(failures, fmt.Errorf("network remove failed: %w", err))
		}
	}
	return errors.Join(failures...)
}

// Use immutable IDs from the container's actual attachments. If the container
// has disappeared, only an explicit ownership label can identify its networks.
func (s *Server) resolveUserRuntimeResources(ctx context.Context, userID string) (string, []string, error) {
	if !ruleset.AllowedUserID(userID) {
		return "", nil, fmt.Errorf("invalid user ID")
	}

	containerName := s.cfg.UserContainerNamePrefix + userID
	inspect, err := s.docker.ContainerInspect(ctx, containerName)
	if err != nil {
		if !errdefs.IsNotFound(err) {
			return "", nil, fmt.Errorf("inspect runtime before cleanup: %w", err)
		}
		networks, err := s.docker.NetworkList(ctx, network.ListOptions{
			Filters: filters.NewArgs(filters.Arg("label", userContainerNetworkLabel+"="+containerName)),
		})
		if err != nil {
			return "", nil, fmt.Errorf("find orphan runtime networks: %w", err)
		}
		var ids []string
		for _, nw := range networks {
			if nw.Labels[userContainerNetworkLabel] == containerName && nw.ID != "" {
				ids = append(ids, nw.ID)
			}
		}
		return "", ids, nil
	}

	var ids []string
	if inspect.NetworkSettings != nil {
		for name, endpoint := range inspect.NetworkSettings.Networks {
			if !strings.HasPrefix(name, s.cfg.NetworkPrefix) || endpoint == nil {
				continue
			}
			if endpoint.NetworkID == "" {
				return "", nil, fmt.Errorf("managed network %s has no ID", name)
			}
			ids = append(ids, endpoint.NetworkID)
		}
	}
	if inspect.ID == "" {
		return "", nil, fmt.Errorf("runtime container has no ID")
	}
	return inspect.ID, ids, nil
}

// disconnectAuthFromUserNetwork detaches the auth container from a user network.
func (s *Server) disconnectAuthFromUserNetwork(ctx context.Context, networkName string) error {
	if err := s.docker.NetworkDisconnect(ctx, networkName, s.cfg.AuthContainerName, true); err != nil {
		if errdefs.IsNotFound(err) || strings.Contains(strings.ToLower(err.Error()), "not found") ||
			strings.Contains(strings.ToLower(err.Error()), "not connected") ||
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
		if errdefs.IsNotFound(err) || strings.Contains(strings.ToLower(err.Error()), "not found") {
			return nil
		}
		return fmt.Errorf("network remove failed: %w", err)
	}
	return nil
}
