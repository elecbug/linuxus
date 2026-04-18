package app

import (
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/elecbug/linuxus/src/ctl/internal/format"
	"github.com/elecbug/linuxus/src/ctl/internal/spec"
)

func (a *App) ensureAuthContainer() error {
	return a.ensureContainer(a.buildAuthRuntimeSpec())
}

func (a *App) ensureManagerContainer() error {
	spec, err := a.buildManagerRuntimeSpec()
	if err != nil {
		return err
	}
	return a.ensureContainer(spec)
}

func (a *App) ensureContainer(spec spec.RuntimeContainerSpec) error {
	cli := a.dockerClient
	if cli == nil {
		return fmt.Errorf("Docker client is not initialized")
	}

	exists, err := a.existDockerContainer(spec.Name)
	if err != nil {
		return err
	}
	if exists {
		fmt.Printf("[+] Recreating container: %s\n", spec.Name)
		if err := cli.ContainerRemove(a.context, spec.Name, container.RemoveOptions{
			Force: true,
		}); err != nil {
			return fmt.Errorf("failed to remove existing container %s: %w", spec.Name, err)
		}
	}

	var (
		exposedPorts nat.PortSet
		portBindings nat.PortMap
	)

	if len(spec.Ports) > 0 {
		exposedPorts = nat.PortSet{}
		portBindings = nat.PortMap{}

		for _, p := range spec.Ports {
			containerPort, hostBinding, err := format.StringToPortBinding(p)
			if err != nil {
				return fmt.Errorf("invalid port binding %q: %w", p, err)
			}
			exposedPorts[containerPort] = struct{}{}
			portBindings[containerPort] = append(portBindings[containerPort], hostBinding)
		}
	}

	cfg := &container.Config{
		Image:        spec.Image,
		Hostname:     spec.Hostname,
		User:         spec.User,
		WorkingDir:   spec.WorkingDir,
		Env:          spec.Environment,
		ExposedPorts: exposedPorts,
	}

	hostCfg := &container.HostConfig{
		Binds:          spec.Volumes,
		Tmpfs:          format.StringsToTmpfsMap(spec.Tmpfs),
		PortBindings:   portBindings,
		ReadonlyRootfs: spec.ReadOnly,
		SecurityOpt:    spec.SecurityOpt,
		CapDrop:        spec.CapDrop,
		RestartPolicy: container.RestartPolicy{
			Name: container.RestartPolicyMode(spec.Restart),
		},
	}

	if spec.Limits.Memory != "" {
		memBytes, err := format.StringToMemoryBytes(spec.Limits.Memory)
		if err != nil {
			return fmt.Errorf("invalid memory limit %q: %w", spec.Limits.Memory, err)
		}
		hostCfg.Memory = memBytes
	}
	if spec.Limits.CPUs != "" {
		nanoCPUs, err := format.StringToNanoCPUs(spec.Limits.CPUs)
		if err != nil {
			return fmt.Errorf("invalid cpu limit %q: %w", spec.Limits.CPUs, err)
		}
		hostCfg.NanoCPUs = nanoCPUs
	}
	if spec.Limits.Pids > 0 {
		pidsLimit := int64(spec.Limits.Pids)
		hostCfg.PidsLimit = &pidsLimit
	}
	if spec.Limits.NofileSoft > 0 || spec.Limits.NofileHard > 0 {
		hostCfg.Ulimits = append(hostCfg.Ulimits, &container.Ulimit{
			Name: "nofile",
			Soft: int64(spec.Limits.NofileSoft),
			Hard: int64(spec.Limits.NofileHard),
		})
	}

	var networkingCfg *network.NetworkingConfig
	if len(spec.Networks) > 0 {
		networkingCfg = &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				spec.Networks[0]: {},
			},
		}
	}

	resp, err := cli.ContainerCreate(
		a.context,
		cfg,
		hostCfg,
		networkingCfg,
		nil,
		spec.Name,
	)
	if err != nil {
		return fmt.Errorf("failed to create container %s: %w", spec.Name, err)
	}

	for _, netName := range spec.Networks[1:] {
		if err := cli.NetworkConnect(a.context, netName, resp.ID, &network.EndpointSettings{}); err != nil {
			return fmt.Errorf("failed to connect %s to %s: %w", spec.Name, netName, err)
		}
	}

	if err := cli.ContainerStart(a.context, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container %s: %w", spec.Name, err)
	}

	return nil
}

func (a *App) removeManagedContainers() error {
	names, err := a.managedContainerNames()
	if err != nil {
		return err
	}

	for _, name := range names {
		exists, err := a.existDockerContainer(name)
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

		fmt.Printf("[+] Removing container: %s\n", name)

		if err := cli.ContainerRemove(a.context, name, container.RemoveOptions{Force: true}); err != nil {
			return fmt.Errorf("failed to remove container %s: %w", name, err)
		}
	}
	return nil
}

func (a *App) managedContainerNames() ([]string, error) {
	cli := a.dockerClient
	if cli == nil {
		return nil, fmt.Errorf("Docker client is not initialized")
	}

	out := []string{
		a.Config.AuthService.Container.Name,
		a.Config.ManagerService.Container.Name,
	}

	summary, err := cli.ContainerList(a.context, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{
		a.Config.AuthService.Container.Name:    {},
		a.Config.ManagerService.Container.Name: {},
	}

	for _, c := range summary {
		for _, raw := range c.Names {
			name := strings.TrimPrefix(raw, "/")
			if strings.HasPrefix(name, a.Config.UserService.Container.NamePrefix) {
				if _, ok := seen[name]; !ok {
					seen[name] = struct{}{}
					out = append(out, name)
				}
			}
		}
	}

	return out, nil
}

func (a *App) existDockerContainer(name string) (bool, error) {
	cli := a.dockerClient
	if cli == nil {
		return false, fmt.Errorf("Docker client is not initialized")
	}

	summary, err := cli.ContainerList(a.context, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.KeyValuePair{Key: "name", Value: "^/" + name + "$"}),
	})
	if err != nil {
		return false, err
	}
	return len(summary) > 0, nil
}
