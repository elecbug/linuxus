package handler

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/elecbug/linuxus/src/internal/common/diskservice"
	"github.com/elecbug/linuxus/src/internal/common/http_helper"
	"github.com/elecbug/linuxus/src/internal/common/packet"
	"github.com/elecbug/linuxus/src/internal/common/ruleset"
	"github.com/elecbug/linuxus/src/internal/common/subnet"
)

// HandleUserUp handles user runtime preparation requests.
func (s *Server) HandleUserUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http_helper.WriteJSONViaHTTP(w, http.StatusMethodNotAllowed, packet.UserUpResponse{
			OK:      false,
			Message: "method not allowed",
		})
		return
	}

	if s.cfg.ManagerSessionSecret != "" && subtle.ConstantTimeCompare(
		[]byte(r.Header.Get("X-Manager-Session-Secret")), []byte(s.cfg.ManagerSessionSecret),
	) != 1 {
		http_helper.WriteJSONViaHTTP(w, http.StatusUnauthorized, packet.UserUpResponse{OK: false, Message: "unauthorized"})
		return
	}
	var req packet.UserUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http_helper.WriteJSONViaHTTP(w, http.StatusBadRequest, packet.UserUpResponse{
			OK:      false,
			Message: "invalid json body",
		})
		return
	}

	req.UserID = strings.TrimSpace(req.UserID)
	if req.UserID == "" {
		http_helper.WriteJSONViaHTTP(w, http.StatusBadRequest, packet.UserUpResponse{
			OK:      false,
			Message: "user_id is required",
		})
		return
	}
	if !ruleset.AllowedUserID(req.UserID) {
		http_helper.WriteJSONViaHTTP(w, http.StatusBadRequest, packet.UserUpResponse{
			OK:      false,
			Message: "user_id contains invalid characters",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.ManagerWaitTime)
	defer cancel()

	resp, err := s.ensureUserRuntimeReady(ctx, req.UserID)
	if err != nil {
		log.Printf("user up failed user=%s err=%v", req.UserID, err)
		http_helper.WriteJSONViaHTTP(w, http.StatusServiceUnavailable, packet.UserUpResponse{
			OK:            false,
			UserID:        req.UserID,
			ContainerName: s.cfg.UserContainerNamePrefix + req.UserID,
			Message:       err.Error(),
		})
		return
	}

	http_helper.WriteJSONViaHTTP(w, http.StatusOK, resp)
}

// ensureUserRuntimeReady ensures a user container and network are ready to serve requests.
func (s *Server) ensureUserRuntimeReady(ctx context.Context, userID string) (response *packet.UserUpResponse, retErr error) {
	if !ruleset.AllowedUserID(userID) {
		return nil, fmt.Errorf("user_id contains invalid characters")
	}

	if s.cfg != nil && s.cfg.MaxPending > 0 {
		if s.pending.Add(1) > int32(s.cfg.MaxPending) {
			s.pending.Add(-1)
			return nil, fmt.Errorf("preparation queue is full; retry shortly")
		}
		defer s.pending.Add(-1)
	}
	if err := s.prepareMu.LockContext(ctx); err != nil {
		return nil, err
	}
	defer s.prepareMu.Unlock()
	defer func() {
		if retErr == nil {
			s.markRuntimeReady(userID)
		}
	}()
	containerName := s.cfg.UserContainerNamePrefix + userID

	_, template, err := s.userTemplate(userID)
	if err != nil {
		return nil, err
	}
	if _, err := s.docker.ImageInspect(ctx, template.Image, client.ImageInspectWithRawResponse(nil)); err != nil {
		return nil, fmt.Errorf("inspect user image %s: %w", template.Image, err)
	}

	exists, running, err := s.inspectContainerState(ctx, containerName)
	if err != nil {
		return nil, err
	}

	if !running {
		if err := s.checkAdmission(ctx); err != nil {
			return nil, err
		}
	}
	if s.cfg.AutoEnsure && !running {
		if err := diskservice.Ensure(ctx, s.diskClient, userID); err != nil {
			return nil, fmt.Errorf("disk preparation failed: %w", err)
		}
	}
	if exists {
		networkName, subnet, err := s.ensureExistingContainerNetworkAndAuth(ctx, containerName)
		if err != nil {
			return nil, err
		}

		if !running {
			if err := s.docker.ContainerStart(ctx, containerName, container.StartOptions{}); err != nil {
				return nil, fmt.Errorf("failed to start existing container: %w", err)
			}
		}

		if _, err := s.waitForContainerIP(ctx, containerName, networkName); err != nil {
			return nil, err
		}

		return &packet.UserUpResponse{
			OK:            true,
			UserID:        userID,
			ContainerName: containerName,
			NetworkName:   networkName,
			Subnet:        subnet,
			Message:       "container ready",
		}, nil
	}

	index, subnet, err := s.findFirstFreeNetworkSlot(ctx)
	if err != nil {
		return nil, err
	}

	networkName := s.cfg.NetworkPrefix + userID
	if exists, err := s.existNetwork(ctx, networkName); err != nil {
		return nil, err
	} else if exists {
		networkName = fmt.Sprintf("%sidx_%d", s.cfg.NetworkPrefix, index)
	}

	networkID, err := s.createNetwork(ctx, networkName, subnet)
	if err != nil {
		return nil, err
	}
	var containerID string
	authConnected := false
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, s.rollbackNewRuntime(ctx, containerID, networkID, authConnected))
		}
	}()
	containerID, err = s.createUserContainer(ctx, containerName, userID, networkName)
	if err != nil {
		return nil, err
	}
	if err := s.ensureAuthConnected(ctx, networkName); err != nil {
		return nil, err
	}
	authConnected = true

	if _, err := s.waitForContainerIP(ctx, containerName, networkName); err != nil {
		return nil, err
	}

	return &packet.UserUpResponse{
		OK:            true,
		UserID:        userID,
		ContainerName: containerName,
		NetworkName:   networkName,
		Subnet:        subnet,
		Message:       "container ready",
	}, nil
}

// inspectContainerState returns existence and running state for a container.
func (s *Server) inspectContainerState(ctx context.Context, name string) (bool, bool, error) {
	inspect, err := s.docker.ContainerInspect(ctx, name)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return false, false, nil
		}
		return false, false, fmt.Errorf("container inspect failed: %w", err)
	}
	if inspect.State != nil && inspect.State.Running {
		return true, true, nil
	}
	return true, false, nil
}

// ensureExistingContainerNetworkAndAuth validates network state for an existing container.
func (s *Server) ensureExistingContainerNetworkAndAuth(ctx context.Context, containerName string) (string, string, error) {
	inspect, err := s.docker.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", "", fmt.Errorf("inspect existing container failed: %w", err)
	}
	if inspect.NetworkSettings == nil || len(inspect.NetworkSettings.Networks) == 0 {
		return "", "", fmt.Errorf("existing container has no network")
	}

	for netName := range inspect.NetworkSettings.Networks {
		if !strings.HasPrefix(netName, s.cfg.NetworkPrefix) {
			continue
		}
		netInfo, err := s.docker.NetworkInspect(ctx, netName, network.InspectOptions{})
		if err != nil {
			return "", "", fmt.Errorf("network inspect failed: %w", err)
		}

		subnet := ""
		if len(netInfo.IPAM.Config) > 0 {
			subnet = strings.TrimSpace(netInfo.IPAM.Config[0].Subnet)
		}

		if err := s.ensureAuthConnected(ctx, netName); err != nil {
			return "", "", err
		}
		return netName, subnet, nil
	}

	return "", "", fmt.Errorf("existing container is not attached to managed network")
}

// ensureAuthConnected attaches auth container to a user network if needed.
func (s *Server) ensureAuthConnected(ctx context.Context, networkName string) error {
	netInfo, err := s.docker.NetworkInspect(ctx, networkName, network.InspectOptions{})
	if err != nil {
		return fmt.Errorf("network inspect failed: %w", err)
	}

	for _, endpoint := range netInfo.Containers {
		if endpoint.Name == s.cfg.AuthContainerName {
			return nil
		}
	}

	if err := s.docker.NetworkConnect(ctx, networkName, s.cfg.AuthContainerName, &network.EndpointSettings{}); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "already exists") ||
			strings.Contains(strings.ToLower(err.Error()), "already connected") {
			return nil
		}
		return fmt.Errorf("failed to connect auth container to %s: %w", networkName, err)
	}

	return nil
}

// createUserContainer creates and starts a user runtime container on the target network.
func (s *Server) createUserContainer(ctx context.Context, containerName, userID, networkName string) (string, error) {
	baseDir := strings.TrimRight(s.cfg.HostHomesDir, "/")
	homeDir := filepath.Clean(baseDir + "/" + userID)
	if !strings.HasPrefix(homeDir, baseDir+"/") {
		return "", fmt.Errorf("invalid user_id: path traversal detected")
	}

	templateName, template, err := s.userTemplate(userID)
	if err != nil {
		return "", err
	}
	cfg := &container.Config{
		Image:      template.Image,
		Hostname:   s.cfg.ContainerHostname,
		User:       s.cfg.RuntimeUser,
		WorkingDir: s.cfg.WorkingDir,
		Env: []string{
			"TZ=" + s.cfg.Timezone,
			"LINUXUS_TEMPLATE=" + templateName,
			"LINUXUS_TEMPLATE_SEED=" + template.Seed,
			"CONTAINER_RUNTIME_USER=" + s.cfg.ContainerRuntimeUser,
			"USER_ID=" + userID,
			"SHARED_DIR=" + s.cfg.ContainerShareDir,
			"READONLY_DIR=" + s.cfg.ContainerReadonlyDir,
			fmt.Sprintf("IS_ADMIN=%t", userID == s.cfg.AdminUserID),
		},
	}

	hostCfg := &container.HostConfig{
		// Missing sources must fail instead of creating an unlimited home directory.
		Mounts: []mount.Mount{
			{Type: mount.TypeBind, Source: homeDir, Target: "/home/" + s.cfg.ContainerRuntimeUser},
			{Type: mount.TypeBind, Source: s.cfg.HostShareDir, Target: s.cfg.ContainerShareDir},
			{Type: mount.TypeBind, Source: s.cfg.HostReadonlyDir, Target: s.cfg.ContainerReadonlyDir, ReadOnly: userID != s.cfg.AdminUserID},
		},
		Tmpfs: map[string]string{
			"/tmp":     "rw,noexec,nosuid,nodev,size=64m",
			"/run":     "rw,noexec,nosuid,nodev,size=16m",
			"/var/tmp": "rw,noexec,nosuid,nodev,size=64m",
		},
		ReadonlyRootfs: s.cfg.ReadOnlyRootFS,
		SecurityOpt:    []string{"no-new-privileges:true"},
		CapDrop:        []string{"ALL"},
		RestartPolicy: container.RestartPolicy{
			Name: "no",
		},
	}

	limits := s.cfg.UserLimits
	if userID == s.cfg.AdminUserID {
		limits = s.cfg.AdminLimits
	}
	if limits.MemoryBytes > 0 {
		hostCfg.Memory = limits.MemoryBytes
	}
	if limits.NanoCPUs > 0 {
		hostCfg.NanoCPUs = limits.NanoCPUs
	}
	if limits.PidsLimit > 0 {
		pids := limits.PidsLimit
		hostCfg.PidsLimit = &pids
	}
	if limits.NofileSoft > 0 && limits.NofileHard > 0 {
		hostCfg.Ulimits = append(hostCfg.Ulimits, &container.Ulimit{
			Name: "nofile",
			Soft: limits.NofileSoft,
			Hard: limits.NofileHard,
		})
	}

	networkingCfg := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkName: {},
		},
	}

	resp, err := s.docker.ContainerCreate(ctx, cfg, hostCfg, networkingCfg, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("container create failed: %w", err)
	}

	if err := s.docker.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return resp.ID, fmt.Errorf("container start failed: %w", err)
	}

	return resp.ID, nil
}

// waitForContainerIP polls until a container obtains an IPv4 on the target network.
func (s *Server) waitForContainerIP(ctx context.Context, containerName, networkName string) (string, error) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		ip, err := s.containerIPv4OnNetwork(ctx, containerName, networkName)
		if err == nil && strings.TrimSpace(ip) != "" {
			return ip, nil
		}

		select {
		case <-ctx.Done():
			return "", fmt.Errorf("container network was not ready within timeout")
		case <-ticker.C:
		}
	}
}

// containerIPv4OnNetwork returns the container IPv4 address on a managed network.
func (s *Server) containerIPv4OnNetwork(ctx context.Context, containerName, networkName string) (string, error) {
	inspect, err := s.docker.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", fmt.Errorf("container inspect failed: %w", err)
	}
	if inspect.NetworkSettings == nil {
		return "", fmt.Errorf("container has no network settings")
	}
	ep, ok := inspect.NetworkSettings.Networks[networkName]
	if !ok || ep == nil {
		return "", fmt.Errorf("container is not attached to network %s", networkName)
	}
	if strings.TrimSpace(ep.IPAddress) == "" {
		return "", fmt.Errorf("container has no ipv4 on network %s", networkName)
	}
	return ep.IPAddress, nil
}

// findFirstFreeNetworkSlot avoids every Docker network's IPAM ranges, including
// unrelated networks and larger ranges that span several user slots.
func (s *Server) findFirstFreeNetworkSlot(ctx context.Context) (int, string, error) {
	if !subnet.IsValidSubnet16(s.cfg.BaseIP) {
		return 0, "", fmt.Errorf("invalid base /16 subnet: %s", s.cfg.BaseIP)
	}
	networks, err := s.docker.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return 0, "", fmt.Errorf("network list failed: %w", err)
	}
	var occupied []*net.IPNet
	for _, nw := range networks {
		for _, cfg := range nw.IPAM.Config {
			raw := strings.TrimSpace(cfg.Subnet)
			if raw == "" {
				continue
			}
			ip, block, err := net.ParseCIDR(raw)
			if err != nil {
				return 0, "", fmt.Errorf("invalid subnet on network %s: %w", nw.Name, err)
			}
			if ip.To4() != nil {
				occupied = append(occupied, block)
			}
		}
	}
	for idx := 0; idx < 4096; idx++ {
		if err := ctx.Err(); err != nil {
			return 0, "", err
		}
		candidate, err := subnet.GetSubnetByIndex(s.cfg.BaseIP, idx)
		if err != nil {
			return 0, "", err
		}
		_, block, _ := net.ParseCIDR(candidate)
		overlaps := false
		for _, used := range occupied {
			if block.Contains(used.IP) || used.Contains(block.IP) {
				overlaps = true
				break
			}
		}
		if !overlaps {
			return idx, candidate, nil
		}
	}
	return 0, "", fmt.Errorf("no free /28 subnet in %s/16", s.cfg.BaseIP)
}

// existNetwork reports whether a Docker network with exact name exists.
func (s *Server) existNetwork(ctx context.Context, name string) (bool, error) {
	nws, err := s.docker.NetworkList(ctx, network.ListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{
			Key:   "name",
			Value: "^" + name + "$",
		}),
	})
	if err != nil {
		return false, fmt.Errorf("network exists query failed: %w", err)
	}
	for _, nw := range nws {
		if nw.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// createNetwork creates a new, isolated network and returns its immutable ID.
func (s *Server) createNetwork(ctx context.Context, name, subnet string) (string, error) {
	if exists, err := s.existNetwork(ctx, name); err != nil {
		return "", err
	} else if exists {
		return "", fmt.Errorf("network name is already in use: %s", name)
	}
	created, err := s.docker.NetworkCreate(ctx, name, network.CreateOptions{
		Driver: "bridge",
		IPAM:   &network.IPAM{Config: []network.IPAMConfig{{Subnet: subnet}}},
	})
	if err != nil {
		return "", fmt.Errorf("network create failed: %w", err)
	}
	return created.ID, nil
}

// rollbackNewRuntime only removes resources created by this request. Cleanup
// still runs if the original preparation context expired or was canceled.
func (s *Server) rollbackNewRuntime(ctx context.Context, containerID, networkID string, authConnected bool) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	var failures []error
	if containerID != "" {
		if err := s.docker.ContainerRemove(cleanupCtx, containerID, container.RemoveOptions{Force: true}); err != nil && !errdefs.IsNotFound(err) {
			failures = append(failures, fmt.Errorf("rollback container %s: %w", containerID, err))
		}
	}
	if networkID != "" {
		if authConnected {
			if err := s.disconnectAuthFromUserNetwork(cleanupCtx, networkID); err != nil {
				failures = append(failures, err)
			}
		}
		if err := s.removeNetwork(cleanupCtx, networkID); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}
