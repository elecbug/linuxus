package app

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func (a *App) GenerateCompose() error {
	fmt.Println("[+] Generating compose file...")

	adminSafe := sanitizeName(a.Config.UserService.Container.Admin.UserID)

	cf := ComposeFile{
		Version:  "3.8",
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

	mountDirs, err := listMountedDirsDeepestFirst(a.Config.Volumes.Host.Homes)
	if err != nil {
		return err
	}

	for _, dir := range mountDirs {
		fmt.Printf("[+] Unmounting: %s\n", dir)
		_ = runCmdAllowFail("sudo", "umount", dir)
	}

	loopDevs, err := findLoopDevicesForImages(a.Config.Volumes.Host.Homes)
	if err != nil {
		return err
	}

	for _, dev := range loopDevs {
		fmt.Printf("[+] Detaching loop device: %s\n", dev)
		_ = runCmdAllowFail("sudo", "losetup", "-d", dev)
	}

	if err := os.RemoveAll(a.Config.Volumes.Host.Homes); err != nil {
		return fmt.Errorf("failed to remove homes dir: %w", err)
	}
	if err := os.RemoveAll(a.Config.Volumes.Host.Share); err != nil {
		return fmt.Errorf("failed to remove share dir: %w", err)
	}
	if err := os.RemoveAll(a.Config.Volumes.Host.Readonly); err != nil {
		return fmt.Errorf("failed to remove readonly dir: %w", err)
	}

	if err := os.MkdirAll(a.Config.Volumes.Host.Homes, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(a.Config.Volumes.Host.Share, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(a.Config.Volumes.Host.Readonly, 0755); err != nil {
		return err
	}

	if err := runCmd(
		"sudo", "chown",
		fmt.Sprintf("%d:%d", a.Config.UserService.Container.Runtime.UID, a.Config.UserService.Container.Runtime.GID),
		a.Config.Volumes.Host.Share,
		a.Config.Volumes.Host.Readonly,
	); err != nil {
		return err
	}

	fmt.Println("[+] Volume clean completed.")
	return nil
}
