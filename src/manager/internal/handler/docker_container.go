package handler

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/elecbug/linuxus/src/manager/internal/config"
)

// createUserContainer creates and starts a user runtime container on the target network.
func (s *Server) createUserContainer(ctx context.Context, containerName, userID, networkName string) error {
	baseDir := strings.TrimRight(s.cfg.HostHomesDir, "/")
	homeDir := filepath.Clean(baseDir + "/" + userID)
	if !strings.HasPrefix(homeDir, baseDir+"/") {
		return fmt.Errorf("invalid user_id: path traversal detected")
	}

	cfg := &container.Config{
		Image:      s.cfg.UserImage,
		Hostname:   s.cfg.ContainerHostname,
		User:       s.cfg.RuntimeUser,
		WorkingDir: s.cfg.WorkingDir,
		Env: []string{
			"TZ=" + s.cfg.Timezone,
			"CONTAINER_RUNTIME_USER=" + s.cfg.ContainerRuntimeUser,
			"USER_ID=" + userID,
			"SHARED_DIR=" + s.cfg.ContainerShareDir,
			"READONLY_DIR=" + s.cfg.ContainerReadonlyDir,
			fmt.Sprintf("IS_ADMIN=%t", userID == s.cfg.AdminUserID),
		},
	}

	hostCfg := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/home/%s:rw", homeDir, s.cfg.ContainerRuntimeUser),
			fmt.Sprintf("%s:%s:rw", s.cfg.HostShareDir, s.cfg.ContainerShareDir),
			getReadonlyBind(userID, s.cfg),
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
		return fmt.Errorf("container create failed: %w", err)
	}

	if err := s.docker.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("container start failed: %w", err)
	}

	return nil
}

// getReadonlyBind returns readonly/shared mount mode based on admin identity.
func getReadonlyBind(userID string, cfg *config.Config) string {
	if userID == cfg.AdminUserID {
		return fmt.Sprintf("%s:%s:rw", cfg.HostReadonlyDir, cfg.ContainerReadonlyDir)
	} else {
		return fmt.Sprintf("%s:%s:ro", cfg.HostReadonlyDir, cfg.ContainerReadonlyDir)
	}
}

func (s *Server) stopAndRemoveUserContainerAndNetwork(ctx context.Context, userID string) error {
	containerName := s.cfg.UserContainerNamePrefix + sanitizeID(userID)
	networkName := s.cfg.NetworkPrefix + sanitizeID(userID)

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
		return fmt.Errorf("container remove failed: %w", err)
	}

	if err := s.disconnectAuthFromNetwork(ctx, networkName); err != nil {
		log.Printf("auth disconnect warning for %s: %v", userID, err)
	}

	if err := s.removeNetwork(ctx, networkName); err != nil {
		return fmt.Errorf("network remove failed: %w", err)
	}

	return nil
}
