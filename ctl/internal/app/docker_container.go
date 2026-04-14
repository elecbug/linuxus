package app

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

func (a *App) ensureAuthContainer() error {
	adminSafe := sanitizeName(a.Config.UserService.Container.Admin.UserID)
	return a.ensureContainer(a.buildAuthRuntimeSpec(adminSafe))
}

func (a *App) ensureUserContainers() error {
	for i := range a.UserIDs {
		if err := a.ensureContainer(a.buildUserRuntimeSpec(a.UserIDs[i], a.SafeIDs[i])); err != nil {
			return err
		}
	}
	adminSafe := sanitizeName(a.Config.UserService.Container.Admin.UserID)
	return a.ensureContainer(a.buildAdminRuntimeSpec(adminSafe))
}

func (a *App) ensureContainer(spec RuntimeContainerSpec) error {
	cli := a.dockerClient
	if cli == nil {
		return fmt.Errorf("Docker client is not initialized")
	}

	exists, err := a.existdockerContainer(spec.Name)
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
			containerPort, hostBinding, err := parsePortBinding(p)
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
		Tmpfs:          parseSliceToTmpfsMap(spec.Tmpfs),
		PortBindings:   portBindings,
		ReadonlyRootfs: spec.ReadOnly,
		SecurityOpt:    spec.SecurityOpt,
		CapDrop:        spec.CapDrop,
		RestartPolicy: container.RestartPolicy{
			Name: container.RestartPolicyMode(spec.Restart),
		},
	}

	if spec.Limits.Memory != "" {
		memBytes, err := parseMemoryBytes(spec.Limits.Memory)
		if err != nil {
			return fmt.Errorf("invalid memory limit %q: %w", spec.Limits.Memory, err)
		}
		hostCfg.Memory = memBytes
	}
	if spec.Limits.CPUs != "" {
		nanoCPUs, err := parseNanoCPUs(spec.Limits.CPUs)
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
	names := a.managedContainerNames()
	for _, name := range names {
		exists, err := a.existdockerContainer(name)
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
