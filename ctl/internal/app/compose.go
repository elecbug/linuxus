package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func (a *App) buildRuntimeImages() error {
	fmt.Println("[+] Building runtime images...")
	if err := runCmd("sudo", "docker", "build", "-t", a.authImageName(), a.Config.AuthService.SourceDir); err != nil {
		return err
	}
	return runCmd(
		"sudo", "docker", "build",
		"--build-arg", "CONTAINER_RUNTIME_USER="+a.Config.UserService.Container.Runtime.User,
		"-t", a.userImageName(),
		a.Config.UserService.SourceDir,
	)
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
	exists, err := dockerNetworkExists(spec.Name)
	if err != nil {
		return err
	}
	if exists {
		fmt.Printf("[=] Network already exists: %s\n", spec.Name)
		return nil
	}
	fmt.Printf("[+] Creating network: %s (%s)\n", spec.Name, spec.Subnet)
	return runCmd("sudo", "docker", "network", "create", "--driver", "bridge", "--subnet", spec.Subnet, spec.Name)
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
	exists, err := dockerContainerExists(spec.Name)
	if err != nil {
		return err
	}
	if exists {
		fmt.Printf("[+] Recreating container: %s\n", spec.Name)
		if err := runCmdAllowFail("sudo", "docker", "rm", "-f", spec.Name); err != nil {
			return fmt.Errorf("failed to remove existing container %s: %w", spec.Name, err)
		}
	}

	args := []string{"docker", "create", "--name", spec.Name}
	if spec.User != "" {
		args = append(args, "--user", spec.User)
	}
	if spec.Hostname != "" {
		args = append(args, "--hostname", spec.Hostname)
	}
	if spec.WorkingDir != "" {
		args = append(args, "--workdir", spec.WorkingDir)
	}
	if spec.ReadOnly {
		args = append(args, "--read-only")
	}
	if spec.Restart != "" {
		args = append(args, "--restart", spec.Restart)
	}
	for _, tmpfs := range spec.Tmpfs {
		args = append(args, "--tmpfs", tmpfs)
	}
	for _, env := range spec.Environment {
		args = append(args, "-e", env)
	}
	for _, volume := range spec.Volumes {
		args = append(args, "-v", volume)
	}
	for _, port := range spec.Ports {
		args = append(args, "-p", port)
	}
	for _, opt := range spec.SecurityOpt {
		args = append(args, "--security-opt", opt)
	}
	for _, cap := range spec.CapDrop {
		args = append(args, "--cap-drop", cap)
	}
	if spec.Limits.Memory != "" {
		args = append(args, "--memory", spec.Limits.Memory)
	}
	if spec.Limits.CPUs != "" {
		args = append(args, "--cpus", spec.Limits.CPUs)
	}
	if spec.Limits.Pids > 0 {
		args = append(args, "--pids-limit", fmt.Sprintf("%d", spec.Limits.Pids))
	}
	if spec.Limits.NofileSoft > 0 || spec.Limits.NofileHard > 0 {
		args = append(args, "--ulimit", fmt.Sprintf("nofile=%d:%d", spec.Limits.NofileSoft, spec.Limits.NofileHard))
	}
	if len(spec.Networks) > 0 {
		args = append(args, "--network", spec.Networks[0])
	}
	args = append(args, spec.Image)

	if err := runCmd("sudo", args...); err != nil {
		return err
	}
	for _, network := range spec.Networks[1:] {
		if err := runCmd("sudo", "docker", "network", "connect", network, spec.Name); err != nil {
			return fmt.Errorf("failed to connect %s to %s: %w", spec.Name, network, err)
		}
	}
	return runCmd("sudo", "docker", "start", spec.Name)
}

func (a *App) removeManagedContainers() error {
	names := a.managedContainerNames()
	for _, name := range names {
		exists, err := dockerContainerExists(name)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		fmt.Printf("[+] Removing container: %s\n", name)
		if err := runCmd("sudo", "docker", "rm", "-f", name); err != nil {
			return err
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
		exists, err := dockerNetworkExists(name)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		fmt.Printf("[+] Removing network: %s\n", name)
		if err := runCmd("sudo", "docker", "network", "rm", name); err != nil {
			return err
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

func dockerContainerExists(name string) (bool, error) {
	out, err := runCmdOutput("sudo", "docker", "ps", "-aq", "--filter", "name=^/"+name+"$")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func dockerNetworkExists(name string) (bool, error) {
	out, err := runCmdOutput("sudo", "docker", "network", "ls", "-q", "--filter", "name=^"+name+"$")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}
