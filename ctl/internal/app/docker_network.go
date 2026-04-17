package app

import (
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
)

// ensureRuntimeNetworks creates required runtime networks if they do not exist.
func (a *App) ensureRuntimeNetworks() error {
	return a.ensureNetwork(RuntimeNetworkSpec{
		Name:   a.Config.ManagerService.Container.Network,
		Subnet: a.Config.ManagerService.Container.Subnet,
	})
}

// ensureNetwork creates a Docker bridge network from the provided specification.
func (a *App) ensureNetwork(spec RuntimeNetworkSpec) error {
	exists, err := a.existDockerNetwork(spec.Name)
	if err != nil {
		return err
	}
	if exists {
		fmt.Printf("[=] Network already exists: %s\n", spec.Name)
		return nil
	}

	cli := a.dockerClient
	if cli == nil {
		return fmt.Errorf("Docker client is not initialized")
	}

	fmt.Printf("[+] Creating network: %s\n", spec.Name)

	_, err = cli.NetworkCreate(a.context, spec.Name, network.CreateOptions{
		Driver: "bridge",
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{
				{
					Subnet: spec.Subnet,
				},
			},
		},
	})
	return err
}

// removeManagedNetworks removes runtime-managed networks in reverse dependency order.
func (a *App) removeManagedNetworks() error {
	names, err := a.managedNetworkNames()
	if err != nil {
		return err
	}

	for i := len(names) - 1; i >= 0; i-- {
		name := names[i]

		exists, err := a.existDockerNetwork(name)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}

		cli := a.dockerClient
		if cli == nil {
			return fmt.Errorf("Docker client is not initialized")
		}

		fmt.Printf("[+] Removing network: %s\n", name)

		if err := cli.NetworkRemove(a.context, name); err != nil {
			return fmt.Errorf("failed to remove network %s: %w", name, err)
		}
	}
	return nil
}

// existDockerNetwork checks whether a network with the exact name exists.
func (a *App) existDockerNetwork(name string) (bool, error) {
	cli := a.dockerClient
	if cli == nil {
		return false, fmt.Errorf("Docker client is not initialized")
	}

	networks, err := cli.NetworkList(a.context, network.ListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{Key: "name", Value: "^" + name + "$"}),
	})
	if err != nil {
		return false, err
	}
	return len(networks) > 0, nil
}

// managedNetworkNames returns network names owned by this runtime manager.
func (a *App) managedNetworkNames() ([]string, error) {
	cli := a.dockerClient
	if cli == nil {
		return nil, fmt.Errorf("Docker client is not initialized")
	}

	out := []string{a.Config.ManagerService.Container.Network}

	networks, err := cli.NetworkList(a.context, network.ListOptions{})
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{
		a.Config.ManagerService.Container.Network: {},
	}

	for _, nw := range networks {
		if strings.HasPrefix(nw.Name, a.Config.UserService.Container.NetworkPrefix) {
			if _, ok := seen[nw.Name]; !ok {
				seen[nw.Name] = struct{}{}
				out = append(out, nw.Name)
			}
		}
	}

	return out, nil
}
