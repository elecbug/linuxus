package app

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/go-connections/nat"
	"github.com/moby/term"
)

func (a *App) GenerateService() error {
	fmt.Println("[+] Preparing runtime service plan...")

	adminSafe := sanitizeName(a.Config.UserService.Container.Admin.UserID)
	networks, err := a.buildRuntimeNetworks()
	if err != nil {
		return err
	}

	fmt.Printf("[=] Auth image: %s\n", a.authImageName())
	fmt.Printf("[=] User image: %s\n", a.userImageName())
	fmt.Printf("[=] Auth container: %s\n", a.buildAuthRuntimeSpec(adminSafe).Name)
	for _, n := range networks {
		fmt.Printf("[=] Network: %s (%s)\n", n.Name, n.Subnet)
	}
	return nil
}

func (a *App) ServiceUp() error {
	fmt.Println("[+] Starting runtime-managed containers...")

	if err := a.buildRuntimeImages(); err != nil {
		return err
	}
	if err := a.ensureRuntimeNetworks(); err != nil {
		return err
	}
	if err := a.ensureAuthContainer(); err != nil {
		return err
	}
	if err := a.ensureUserContainers(); err != nil {
		return err
	}

	fmt.Println("[+] Runtime services started.")
	return nil
}

func (a *App) ServiceDown() error {
	fmt.Println("[+] Stopping runtime-managed containers...")
	if err := a.removeManagedContainers(); err != nil {
		return err
	}
	if err := a.removeManagedNetworks(); err != nil {
		return err
	}
	return nil
}

func (a *App) ServiceRestart() error {
	fmt.Println("[+] Restarting runtime-managed containers...")
	if err := a.ServiceDown(); err != nil {
		return err
	}
	return a.ServiceUp()
}

func (a *App) VolumeClean() error {
	fmt.Println("[+] Cleaning volumes...")

	_ = a.ServiceDown()

	homeMounts, err := listMountedDirsDeepestFirst(a.Config.Volumes.Host.Homes)
	if err != nil {
		return err
	}
	for _, dir := range homeMounts {
		fmt.Printf("[+] Unmounting: %s\n", dir)
		_ = runCmdAllowFail("sudo", "umount", dir)
	}

	for _, mountPoint := range []string{a.Config.Volumes.Host.Share, a.Config.Volumes.Host.Readonly} {
		if mounted, err := isMountPoint(mountPoint); err == nil && mounted {
			fmt.Printf("[+] Unmounting: %s\n", mountPoint)
			_ = runCmdAllowFail("sudo", "umount", mountPoint)
		}
	}

	homeDevs, err := findLoopDevicesForImages(a.Config.Volumes.Host.Homes)
	if err != nil {
		return err
	}

	seen := make(map[string]struct{})
	var loopDevs []string
	for _, dev := range homeDevs {
		if _, exists := seen[dev]; !exists {
			seen[dev] = struct{}{}
			loopDevs = append(loopDevs, dev)
		}
	}
	for _, mountPoint := range []string{a.Config.Volumes.Host.Share, a.Config.Volumes.Host.Readonly} {
		devs, err := findLoopDevicesForImages(filepath.Dir(mountPoint))
		if err != nil {
			return err
		}
		for _, dev := range devs {
			if _, exists := seen[dev]; !exists {
				seen[dev] = struct{}{}
				loopDevs = append(loopDevs, dev)
			}
		}
	}

	for _, dev := range loopDevs {
		fmt.Printf("[+] Detaching loop device: %s\n", dev)
		_ = runCmdAllowFail("sudo", "losetup", "-d", dev)
	}

	if err := os.RemoveAll(a.Config.Volumes.Host.Homes); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove homes dir: %w", err)
	}
	if err := os.RemoveAll(a.Config.Volumes.Host.Share); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove share dir: %w", err)
	}
	if err := os.RemoveAll(a.Config.Volumes.Host.Readonly); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove readonly dir: %w", err)
	}
	if err := os.RemoveAll(a.Config.Volumes.Host.Volumes); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove volumes dir: %w", err)
	}

	fmt.Println("[+] Volume clean completed.")
	return nil
}
func (a *App) ServicePS() error {
	fmt.Println("[+] Runtime service status:")

	names := make([]string, 0, len(a.UserIDs)+2)
	names = append(names, a.Config.AuthService.Container.Name)

	for _, id := range a.UserIDs {
		names = append(names, a.Config.UserService.Container.NamePrefix+id)
	}

	adminID := sanitizeName(a.Config.UserService.Container.Admin.UserID)
	names = append(names, a.Config.UserService.Container.NamePrefix+adminID)

	fmt.Printf("%-24s %-12s %-20s %-25s %s\n", "NAME", "STATE", "STATUS", "IMAGE", "PORTS")

	for _, name := range names {
		info, err := a.DockerClient.ContainerInspect(a.Context, name)
		if err != nil {
			if errdefs.IsNotFound(err) {
				fmt.Printf("%-24s %-12s %-20s %-25s %s\n", name, "missing", "not created", "-", "-")
				continue
			}
			return fmt.Errorf("failed to inspect container %s: %w", name, err)
		}

		state := "-"
		status := "-"
		if info.State != nil {
			state = info.State.Status
			status = containerStatusText(info)
		}

		image := info.Config.Image
		ports := portSummary(info)

		fmt.Printf("%-24s %-12s %-20s %-25s %s\n", name, state, status, image, ports)
	}

	return nil
}

func containerStatusText(info types.ContainerJSON) string {
	if info.State == nil {
		return "-"
	}

	status := info.State.Status

	if status == "exited" {
		return fmt.Sprintf("exited(%d)", info.State.ExitCode)
	}

	if info.State.OOMKilled {
		return "oom-killed"
	}

	return status
}

func portSummary(info types.ContainerJSON) string {
	if info.NetworkSettings == nil || len(info.NetworkSettings.Ports) == 0 {
		return "-"
	}

	first := true
	out := ""

	for containerPort, bindings := range info.NetworkSettings.Ports {
		if len(bindings) == 0 {
			if !first {
				out += ", "
			}
			out += string(containerPort)
			first = false
			continue
		}

		for _, b := range bindings {
			if !first {
				out += ", "
			}
			if b.HostIP != "" {
				out += fmt.Sprintf("%s:%s->%s", b.HostIP, b.HostPort, containerPort)
			} else {
				out += fmt.Sprintf("%s->%s", b.HostPort, containerPort)
			}
			first = false
		}
	}

	if out == "" {
		return "-"
	}
	return out
}

func (a *App) buildRuntimeImages() error {
	fmt.Println("[+] Building runtime images...")

	if a.DockerClient == nil {
		return fmt.Errorf("Docker client is not initialized")
	}

	if err := a.buildImage(a.Config.AuthService.SourceDir, a.authImageName(), nil); err != nil {
		return fmt.Errorf("failed to build auth image: %w", err)
	}

	if err := a.buildImage(a.Config.UserService.SourceDir, a.userImageName(), map[string]*string{
		"CONTAINER_RUNTIME_USER": strPtr(a.Config.UserService.Container.Runtime.User),
	}); err != nil {
		return fmt.Errorf("failed to build user image: %w", err)
	}

	return nil
}

func (a *App) buildImage(sourceDir string, tag string, buildArgs map[string]*string) error {
	buildCtx, err := tarBuildContext(sourceDir)
	if err != nil {
		return err
	}

	resp, err := a.DockerClient.ImageBuild(a.Context, buildCtx, build.ImageBuildOptions{
		Tags:       []string{tag},
		Dockerfile: "Dockerfile",
		Remove:     true,
		BuildArgs:  buildArgs,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fd, isTerm := term.GetFdInfo(os.Stdout)

	err = jsonmessage.DisplayJSONMessagesStream(
		resp.Body,
		os.Stdout,
		fd,
		isTerm,
		nil,
	)
	if err != nil {
		return err
	}

	return nil
}

func tarBuildContext(dir string) (io.Reader, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		_ = tw.Close()
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return buf, nil
}

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
	exists, err := a.dockerNetworkExists(spec.Name)
	if err != nil {
		return err
	}
	if exists {
		fmt.Printf("[=] Network already exists: %s\n", spec.Name)
		return nil
	}

	cli := a.DockerClient
	if cli == nil {
		return fmt.Errorf("Docker client is not initialized")
	}

	fmt.Printf("[+] Creating network: %s (%s)\n", spec.Name, spec.Subnet)

	_, err = cli.NetworkCreate(a.Context, spec.Name, network.CreateOptions{
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
	cli := a.DockerClient
	if cli == nil {
		return fmt.Errorf("Docker client is not initialized")
	}

	exists, err := a.dockerContainerExists(spec.Name)
	if err != nil {
		return err
	}
	if exists {
		fmt.Printf("[+] Recreating container: %s\n", spec.Name)
		if err := cli.ContainerRemove(a.Context, spec.Name, container.RemoveOptions{
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
		Tmpfs:          sliceToTmpfsMap(spec.Tmpfs),
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
		a.Context,
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
		if err := cli.NetworkConnect(a.Context, netName, resp.ID, &network.EndpointSettings{}); err != nil {
			return fmt.Errorf("failed to connect %s to %s: %w", spec.Name, netName, err)
		}
	}

	if err := cli.ContainerStart(a.Context, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container %s: %w", spec.Name, err)
	}

	return nil
}

func (a *App) removeManagedContainers() error {
	names := a.managedContainerNames()
	for _, name := range names {
		exists, err := a.dockerContainerExists(name)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}

		cli := a.DockerClient
		if cli == nil {
			return fmt.Errorf("Docker client is not initialized")
		}

		fmt.Printf("[+] Removing container: %s\n", name)

		if err := cli.ContainerRemove(a.Context, name, container.RemoveOptions{Force: true}); err != nil {
			return fmt.Errorf("failed to remove container %s: %w", name, err)
		}
	}
	return nil
}

func (a *App) removeManagedNetworks() error {
	networks, err := a.buildRuntimeNetworks()
	if err != nil {
		return err
	}
	for i := len(networks) - 1; i >= 0; i-- {
		name := networks[i].Name
		exists, err := a.dockerNetworkExists(name)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}

		cli := a.DockerClient
		if cli == nil {
			return fmt.Errorf("Docker client is not initialized")
		}

		fmt.Printf("[+] Removing network: %s\n", name)

		if err := cli.NetworkRemove(a.Context, name); err != nil {
			return fmt.Errorf("failed to remove network %s: %w", name, err)
		}
	}
	return nil
}

func (a *App) managedContainerNames() []string {
	names := make([]string, 0, len(a.SafeIDs)+2)
	names = append(names, a.Config.AuthService.Container.Name)
	for _, safeID := range a.SafeIDs {
		names = append(names, a.Config.UserService.Container.NamePrefix+safeID)
	}
	names = append(names, a.Config.UserService.Container.NamePrefix+a.Config.UserService.Container.Admin.UserID)
	return names
}

func (a *App) dockerContainerExists(name string) (bool, error) {
	cli := a.DockerClient
	if cli == nil {
		return false, fmt.Errorf("Docker client is not initialized")
	}

	summary, err := cli.ContainerList(a.Context, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.KeyValuePair{Key: "name", Value: "^/" + name + "$"}),
	})
	if err != nil {
		return false, err
	}
	return len(summary) > 0, nil
}

func (a *App) dockerNetworkExists(name string) (bool, error) {
	cli := a.DockerClient
	if cli == nil {
		return false, fmt.Errorf("Docker client is not initialized")
	}

	networks, err := cli.NetworkList(a.Context, network.ListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{Key: "name", Value: "^" + name + "$"}),
	})
	if err != nil {
		return false, err
	}
	return len(networks) > 0, nil
}

func (a *App) buildAuthRuntimeSpec(adminSafe string) RuntimeContainerSpec {
	networks := make([]string, 0, len(a.SafeIDs)+1)
	for _, safeID := range a.SafeIDs {
		networks = append(networks, a.Config.UserService.Container.NetworkPrefix+safeID)
	}
	networks = append(networks, a.Config.UserService.Container.NetworkPrefix+adminSafe)

	return RuntimeContainerSpec{
		Image: a.authImageName(),
		Name:  a.Config.AuthService.Container.Name,
		Environment: []string{
			"TZ=" + a.Config.AuthService.Container.Timezone,
			"AUTH_LIST=" + a.Config.AuthService.AuthListFile.ContainerPath,
			"SESSION_SECRET=" + a.Config.AuthService.Security.SessionSecret,
			"LOGIN_PATH=" + a.Config.AuthService.URLPath.Login,
			"LOGOUT_PATH=" + a.Config.AuthService.URLPath.Logout,
			"SERVICE_PATH=" + a.Config.AuthService.URLPath.Service,
			"TERMINAL_PATH=" + a.Config.AuthService.URLPath.Terminal,
			"USER_CONTAINER_NAME_PREFIX=" + a.Config.UserService.Container.NamePrefix,
			"TRUSTED_PROXIES=" + a.Config.AuthService.Security.TrustedProxies,
		},
		Volumes: []string{
			fmt.Sprintf("%s:%s:rw", a.Config.AuthService.AuthListFile.HostPath, a.Config.AuthService.AuthListFile.ContainerPath),
		},
		Ports: []string{
			fmt.Sprintf("%d:8080", a.Config.AuthService.Container.ExternalPort),
		},
		Restart:  "unless-stopped",
		Networks: networks,
	}
}

func (a *App) buildUserRuntimeSpec(userID, safeID string) RuntimeContainerSpec {
	return RuntimeContainerSpec{
		Image:      a.userImageName(),
		Name:       a.Config.UserService.Container.NamePrefix + safeID,
		User:       fmt.Sprintf("%d:%d", a.Config.UserService.Container.Runtime.UID, a.Config.UserService.Container.Runtime.GID),
		Hostname:   a.Config.UserService.Container.Runtime.Hostname,
		WorkingDir: "/home/" + a.Config.UserService.Container.Runtime.User,
		ReadOnly:   true,
		Tmpfs: []string{
			"/tmp:rw,noexec,nosuid,nodev,size=64m",
			"/run:rw,noexec,nosuid,nodev,size=16m",
			"/var/tmp:rw,noexec,nosuid,nodev,size=64m",
		},
		Environment: []string{
			"TZ=" + a.Config.UserService.Container.Runtime.Timezone,
			"CONTAINER_RUNTIME_USER=" + a.Config.UserService.Container.Runtime.User,
			"USER_ID=" + userID,
			"SHARED_DIR=" + a.Config.Volumes.Container.Share,
			"READONLY_DIR=" + a.Config.Volumes.Container.Readonly,
			"IS_ADMIN=false",
		},
		Volumes: []string{
			fmt.Sprintf("%s:/home/%s:rw", a.homeDirForUser(userID), a.Config.UserService.Container.Runtime.User),
			fmt.Sprintf("%s:%s:rw", a.Config.Volumes.Host.Share, a.Config.Volumes.Container.Share),
			fmt.Sprintf("%s:%s:ro", a.Config.Volumes.Host.Readonly, a.Config.Volumes.Container.Readonly),
		},
		Restart:     "unless-stopped",
		SecurityOpt: []string{"no-new-privileges:true"},
		CapDrop:     []string{"ALL"},
		Limits: ContainerLimits{
			Memory:     a.Config.UserService.Container.User.Limits.Memory,
			CPUs:       fmt.Sprintf("%v", a.Config.UserService.Container.User.Limits.CPU),
			Pids:       a.Config.UserService.Container.User.Limits.PID,
			NofileSoft: a.Config.UserService.Container.User.Limits.Ulimits.Nofile.Soft,
			NofileHard: a.Config.UserService.Container.User.Limits.Ulimits.Nofile.Hard,
		},
		Networks: []string{
			a.Config.UserService.Container.NetworkPrefix + safeID,
		},
	}
}

func (a *App) buildAdminRuntimeSpec(adminSafe string) RuntimeContainerSpec {
	return RuntimeContainerSpec{
		Image:      a.userImageName(),
		Name:       a.Config.UserService.Container.NamePrefix + a.Config.UserService.Container.Admin.UserID,
		User:       fmt.Sprintf("%d:%d", a.Config.UserService.Container.Runtime.UID, a.Config.UserService.Container.Runtime.GID),
		Hostname:   a.Config.UserService.Container.Runtime.Hostname,
		WorkingDir: "/home/" + a.Config.UserService.Container.Runtime.User,
		ReadOnly:   true,
		Tmpfs: []string{
			"/tmp:rw,noexec,nosuid,nodev,size=64m",
			"/run:rw,noexec,nosuid,nodev,size=16m",
			"/var/tmp:rw,noexec,nosuid,nodev,size=64m",
		},
		Environment: []string{
			"TZ=" + a.Config.UserService.Container.Runtime.Timezone,
			"CONTAINER_RUNTIME_USER=" + a.Config.UserService.Container.Runtime.User,
			"USER_ID=" + a.Config.UserService.Container.Admin.UserID,
			"SHARED_DIR=" + a.Config.Volumes.Container.Share,
			"READONLY_DIR=" + a.Config.Volumes.Container.Readonly,
			"IS_ADMIN=true",
		},
		Volumes: []string{
			fmt.Sprintf("%s:/home/%s:rw", a.homeDirForUser(a.Config.UserService.Container.Admin.UserID), a.Config.UserService.Container.Runtime.User),
			fmt.Sprintf("%s:%s:rw", a.Config.Volumes.Host.Share, a.Config.Volumes.Container.Share),
			fmt.Sprintf("%s:%s:rw", a.Config.Volumes.Host.Readonly, a.Config.Volumes.Container.Readonly),
		},
		Restart:     "unless-stopped",
		SecurityOpt: []string{"no-new-privileges:true"},
		CapDrop:     []string{"ALL"},
		Limits: ContainerLimits{
			Memory:     a.Config.UserService.Container.Admin.Limits.Memory,
			CPUs:       fmt.Sprintf("%v", a.Config.UserService.Container.Admin.Limits.CPU),
			Pids:       a.Config.UserService.Container.Admin.Limits.PID,
			NofileSoft: a.Config.UserService.Container.Admin.Limits.Ulimits.Nofile.Soft,
			NofileHard: a.Config.UserService.Container.Admin.Limits.Ulimits.Nofile.Hard,
		},
		Networks: []string{
			a.Config.UserService.Container.NetworkPrefix + adminSafe,
		},
	}
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

func parsePortBinding(s string) (nat.Port, nat.PortBinding, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return "", nat.PortBinding{}, fmt.Errorf("expected HOST:CONTAINER, got %q", s)
	}

	hostPort := strings.TrimSpace(parts[0])
	containerPart := strings.TrimSpace(parts[1])

	containerPort, err := nat.NewPort("tcp", containerPart)
	if err != nil {
		return "", nat.PortBinding{}, err
	}

	return containerPort, nat.PortBinding{
		HostIP:   "",
		HostPort: hostPort,
	}, nil
}

func sliceToTmpfsMap(items []string) map[string]string {
	if len(items) == 0 {
		return nil
	}

	out := make(map[string]string, len(items))
	for _, item := range items {
		parts := strings.SplitN(item, ":", 2)
		mountPoint := strings.TrimSpace(parts[0])
		if mountPoint == "" {
			continue
		}

		opts := ""
		if len(parts) == 2 {
			opts = strings.TrimSpace(parts[1])
		}
		out[mountPoint] = opts
	}
	return out
}

func parseNanoCPUs(v string) (int64, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return 0, err
	}
	if f < 0 {
		return 0, fmt.Errorf("must be non-negative")
	}
	return int64(f * 1_000_000_000), nil
}

func parseMemoryBytes(v string) (int64, error) {
	s := strings.TrimSpace(strings.ToLower(v))
	mult := int64(1)

	switch {
	case strings.HasSuffix(s, "g"):
		mult = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "g")
	case strings.HasSuffix(s, "gb"):
		mult = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "gb")
	case strings.HasSuffix(s, "m"):
		mult = 1024 * 1024
		s = strings.TrimSuffix(s, "m")
	case strings.HasSuffix(s, "mb"):
		mult = 1024 * 1024
		s = strings.TrimSuffix(s, "mb")
	case strings.HasSuffix(s, "k"):
		mult = 1024
		s = strings.TrimSuffix(s, "k")
	case strings.HasSuffix(s, "kb"):
		mult = 1024
		s = strings.TrimSuffix(s, "kb")
	case strings.HasSuffix(s, "b"):
		mult = 1
		s = strings.TrimSuffix(s, "b")
	}

	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, fmt.Errorf("must be non-negative")
	}
	return n * mult, nil
}
