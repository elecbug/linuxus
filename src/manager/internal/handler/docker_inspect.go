package handler

import (
	"context"
	"fmt"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func (s *Server) ensureUserRuntimeReady(ctx context.Context, userID, safeID string) (*UserUpResponse, error) {
	containerName := s.cfg.UserContainerNamePrefix + safeID

	if _, err := s.docker.ImageInspect(ctx, s.cfg.UserImage, client.ImageInspectWithRawResponse(nil)); err != nil {
		return nil, fmt.Errorf("user image not found: %s", s.cfg.UserImage)
	}

	exists, running, err := s.inspectContainerState(ctx, containerName)
	if err != nil {
		return nil, err
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

		return &UserUpResponse{
			OK:            true,
			UserID:        userID,
			SafeID:        safeID,
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

	networkName := s.cfg.NetworkPrefix + safeID
	if exists, err := s.existNetwork(ctx, networkName); err != nil {
		return nil, err
	} else if exists {
		networkName = fmt.Sprintf("%sidx_%d", s.cfg.NetworkPrefix, index)
	}

	if err := s.createNetwork(ctx, networkName, subnet); err != nil {
		return nil, err
	}

	if err := s.createUserContainer(ctx, containerName, userID, networkName); err != nil {
		return nil, err
	}

	if err := s.ensureAuthConnected(ctx, networkName); err != nil {
		return nil, err
	}

	if _, err := s.waitForContainerIP(ctx, containerName, networkName); err != nil {
		return nil, err
	}

	return &UserUpResponse{
		OK:            true,
		UserID:        userID,
		SafeID:        safeID,
		ContainerName: containerName,
		NetworkName:   networkName,
		Subnet:        subnet,
		Message:       "container ready",
	}, nil
}

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
