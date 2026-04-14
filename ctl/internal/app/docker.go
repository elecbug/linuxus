package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/containerd/errdefs"
)

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

	names := a.managedContainerNames()

	results := make([]containerInfo, 0, len(names)+1)
	results = append(results, containerInfo{
		Name:   "NAME",
		State:  "STATE",
		Status: "STATUS",
		Image:  "IMAGE",
		Ports:  "PORTS",
	})

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
			status = parseContainerStatusText(info)
		}

		image := info.Config.Image
		ports := parsePortSummary(info)

		results = append(results, containerInfo{
			Name:   name,
			State:  state,
			Status: status,
			Image:  image,
			Ports:  ports,
		})
	}

	strResults := parseContainerInfos(results)

	for _, result := range strResults {
		fmt.Println(result)
	}

	return nil
}
