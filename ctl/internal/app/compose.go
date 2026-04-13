package app

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func (a *App) GenerateCompose() error {
	fmt.Println("[+] Generating compose file...")

	adminSafe := sanitizeName(a.Config.UserService.Container.Admin.UserID)

	cf := ComposeFile{
		Services: make(map[string]ComposeService),
		Networks: make(map[string]ComposeNetwork),
	}

	authServiceName := a.Config.AuthService.Container.Name
	cf.Services[authServiceName] = a.buildAuthService(adminSafe)

	for i := range a.UserIDs {
		serviceName := a.Config.UserService.Container.NamePrefix + a.SafeIDs[i]
		cf.Services[serviceName] = a.buildUserService(a.UserIDs[i], a.SafeIDs[i])
	}

	adminServiceName := a.Config.UserService.Container.NamePrefix + a.Config.UserService.Container.Admin.UserID
	cf.Services[adminServiceName] = a.buildAdminService(adminSafe)

	seq := 0
	for _, safeID := range a.SafeIDs {
		networkName := a.Config.UserService.Container.NetworkPrefix + safeID
		subnet, err := getIP(a.Config.UserService.Container.BaseIP, seq)
		if err != nil {
			return err
		}
		cf.Networks[networkName] = ComposeNetwork{
			Driver: "bridge",
			IPAM: &ComposeIPAM{
				Config: []ComposeSubnet{{Subnet: subnet}},
			},
		}
		seq++
	}

	adminNetworkName := a.Config.UserService.Container.NetworkPrefix + adminSafe
	subnet, err := getIP(a.Config.UserService.Container.BaseIP, seq)
	if err != nil {
		return err
	}
	cf.Networks[adminNetworkName] = ComposeNetwork{
		Driver: "bridge",
		IPAM: &ComposeIPAM{
			Config: []ComposeSubnet{{Subnet: subnet}},
		},
	}

	data, err := yaml.Marshal(&cf)
	if err != nil {
		return fmt.Errorf("failed to marshal compose yaml: %w", err)
	}

	if err := os.WriteFile(a.Config.Compose.OutputFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

func (a *App) ComposeUp() error {
	fmt.Println("[+] Starting containers...")
	return runCmd("sudo", "docker", "compose", "-f", a.Config.Compose.OutputFile, "up", "-d", "--build")
}

func (a *App) ComposeDown() error {
	fmt.Println("[+] Stopping containers...")
	return runCmd("sudo", "docker", "compose", "-f", a.Config.Compose.OutputFile, "down", "--remove-orphans")
}

func (a *App) ComposeRestart() error {
	fmt.Println("[+] Restarting containers...")
	if err := runCmd("sudo", "docker", "compose", "-f", a.Config.Compose.OutputFile, "down", "--remove-orphans"); err != nil {
		return err
	}
	return runCmd("sudo", "docker", "compose", "-f", a.Config.Compose.OutputFile, "up", "-d", "--build")
}

func (a *App) VolumeClean() error {
	fmt.Println("[+] Cleaning volumes...")

	_ = runCmdAllowFail("sudo", "docker", "compose", "-f", a.Config.Compose.OutputFile, "down", "-v", "--remove-orphans")

	// Unmount home directories (each user home is a mount point under the homes root).
	homeMounts, err := listMountedDirsDeepestFirst(a.Config.Volumes.Host.Homes)
	if err != nil {
		return err
	}
	for _, dir := range homeMounts {
		fmt.Printf("[+] Unmounting: %s\n", dir)
		_ = runCmdAllowFail("sudo", "umount", dir)
	}

	// Unmount share and readonly mount points directly, regardless of whether
	// they reside under volumes.host.volumes.
	for _, mountPoint := range []string{a.Config.Volumes.Host.Share, a.Config.Volumes.Host.Readonly} {
		if mounted, err := isMountPoint(mountPoint); err == nil && mounted {
			fmt.Printf("[+] Unmounting: %s\n", mountPoint)
			_ = runCmdAllowFail("sudo", "umount", mountPoint)
		}
	}

	// Find loop devices for home disk images.
	homeDevs, err := findLoopDevicesForImages(a.Config.Volumes.Host.Homes)
	if err != nil {
		return err
	}

	// Find loop devices for share and readonly disk images.  Each image is
	// stored as <name>.img in the parent directory of the mount point, so scan
	// filepath.Dir(Share) and filepath.Dir(Readonly) rather than the volumes
	// root, ensuring images outside that root are also found.
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
