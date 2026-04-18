package handler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
)

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

func (s *Server) disconnectAuthFromNetwork(ctx context.Context, networkName string) error {
	if err := s.docker.NetworkDisconnect(ctx, networkName, s.cfg.AuthContainerName, true); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") ||
			strings.Contains(strings.ToLower(err.Error()), "already disconnected") {
			return nil
		}
		return fmt.Errorf("failed to disconnect auth container from %s: %w", networkName, err)
	}

	return nil
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

func (s *Server) findFirstFreeNetworkSlot(ctx context.Context) (int, string, error) {
	networks, err := s.docker.NetworkList(ctx, network.ListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{
			Key:   "name",
			Value: "^" + s.cfg.NetworkPrefix,
		}),
	})
	if err != nil {
		return 0, "", fmt.Errorf("network list failed: %w", err)
	}

	used := make(map[int]struct{})

	for _, nw := range networks {
		if !strings.HasPrefix(nw.Name, s.cfg.NetworkPrefix) {
			continue
		}

		inspect, err := s.docker.NetworkInspect(ctx, nw.ID, network.InspectOptions{})
		if err != nil {
			return 0, "", fmt.Errorf("network inspect failed: %w", err)
		}
		if len(inspect.IPAM.Config) == 0 {
			continue
		}

		subnet := strings.TrimSpace(inspect.IPAM.Config[0].Subnet)
		idx, ok := subnetToIndex(s.cfg.BaseIP, subnet)
		if ok {
			used[idx] = struct{}{}
		}
	}

	for idx := 0; ; idx++ {
		if _, exists := used[idx]; exists {
			continue
		}
		subnet, err := getSubnetByIndex(s.cfg.BaseIP, idx)
		if err != nil {
			return 0, "", err
		}
		return idx, subnet, nil
	}
}

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

func (s *Server) createNetwork(ctx context.Context, name, subnet string) error {
	if exists, err := s.existNetwork(ctx, name); err != nil {
		return err
	} else if exists {
		return nil
	}

	_, err := s.docker.NetworkCreate(ctx, name, network.CreateOptions{
		Driver: "bridge",
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{
				{Subnet: subnet},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("network create failed: %w", err)
	}
	return nil
}

func (s *Server) removeNetwork(ctx context.Context, name string) error {
	if err := s.docker.NetworkRemove(ctx, name); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return nil
		}
		return fmt.Errorf("network remove failed: %w", err)
	}
	return nil
}
