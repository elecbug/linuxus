package app

import (
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/elecbug/linuxus/src/ctl/internal/spec"
)

// ensureRuntimeNetworks creates required runtime networks.
func (a *App) ensureRuntimeNetworks() error {
	return a.ensureNetwork(spec.RuntimeNetworkSpec{
		Name:   a.Config.ManagerService.Container.Network,
		Subnet: a.Config.ManagerService.Container.Subnet,
	})
}

// ensureNetwork creates a Docker network if it does not already exist.
func (a *App) ensureNetwork(spec spec.RuntimeNetworkSpec) error {
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

// removeManagedNetworks removes all networks controlled by this CLI.
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

// managedNetworkNames returns known managed network names, including user networks.
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
		if strings.HasPrefix(nw.Name, a.Config.UserService.Container.NetworkNamePrefix) {
			if _, ok := seen[nw.Name]; !ok {
				seen[nw.Name] = struct{}{}
				out = append(out, nw.Name)
			}
		}
	}

	return out, nil
}

// existDockerNetwork checks whether a network with exact name exists.
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
