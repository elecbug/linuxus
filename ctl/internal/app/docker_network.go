package app

import (
	"fmt"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
)

func (a *App) ensureRuntimeNetworks() error {
	networks, err := a.buildRuntimeNetworks()
	if err != nil {
		return err
	}
	for _, network := range networks {
		if err := a.ensureNetwork(network); err != nil {
			return err
		}
	}
	return nil
}

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

	fmt.Printf("[+] Creating network: %s (%s)\n", spec.Name, spec.Subnet)

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

func (a *App) removeManagedNetworks() error {
	networks, err := a.buildRuntimeNetworks()
	if err != nil {
		return err
	}
	for i := len(networks) - 1; i >= 0; i-- {
		name := networks[i].Name
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

func (a *App) buildRuntimeNetworks() ([]RuntimeNetworkSpec, error) {
	networks := make([]RuntimeNetworkSpec, 0, len(a.SafeIDs)+1)
	seq := 0

	for _, safeID := range a.SafeIDs {
		subnet, err := getIP(a.Config.UserService.Container.BaseIP, seq)
		if err != nil {
			return nil, err
		}
		networks = append(networks, RuntimeNetworkSpec{
			Name:   a.Config.UserService.Container.NetworkPrefix + safeID,
			Subnet: subnet,
		})
		seq++
	}

	adminSafe := sanitizeName(a.Config.UserService.Container.Admin.UserID)
	subnet, err := getIP(a.Config.UserService.Container.BaseIP, seq)
	if err != nil {
		return nil, err
	}
	networks = append(networks, RuntimeNetworkSpec{
		Name:   a.Config.UserService.Container.NetworkPrefix + adminSafe,
		Subnet: subnet,
	})

	return networks, nil
}
