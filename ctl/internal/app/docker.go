package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/network"
)

// ServiceUp builds images and ensures all runtime-managed services are running.
func (a *App) ServiceUp() error {
	fmt.Println("[+] Starting runtime-managed containers...")

	if err := a.buildRuntimeImages(); err != nil {
		return err
	}
	if err := a.ensureRuntimeNetworks(); err != nil {
		return err
	}
	if err := a.ensureManagerContainer(); err != nil {
		return err
	}
	if err := a.ensureAuthContainer(); err != nil {
		return err
	}

	fmt.Println("[+] Runtime services started.")
	return nil
}

// ServiceDown removes runtime-managed containers and networks.
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

// ServiceRestart restarts runtime-managed services by running down then up.
func (a *App) ServiceRestart() error {
	fmt.Println("[+] Restarting runtime-managed containers...")
	if err := a.ServiceDown(); err != nil {
		return err
	}
	return a.ServiceUp()
}

// VolumeClean stops services, unmounts disks, detaches loops, and removes volume paths.
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

// ServicePS prints table-form runtime status for managed containers and networks.
func (a *App) ServicePS() error {
	fmt.Println("[+] Runtime service status:")

	names, err := a.managedContainerNames()
	if err != nil {
		return err
	}

	containerInfos := make([]containerInfo, 0, len(names)+1)
	containerInfos = append(containerInfos, containerInfo{
		Name:   "CONTAINER NAME",
		Status: "STATE(STATUS)",
		Image:  "IMAGE",
		Ports:  "PORTS",
		UserID: "USER ID",
	})

	for _, name := range names {
		info, err := a.dockerClient.ContainerInspect(a.context, name)
		if err != nil {
			if errdefs.IsNotFound(err) {
				containerInfos = append(containerInfos, containerInfo{
					Name:   name,
					Status: "not found",
					Image:  "-",
					Ports:  "-",
					UserID: "-",
				})
				continue
			}
			return fmt.Errorf("failed to inspect container %s: %w", name, err)
		}

		hasState := false
		state := "-"
		status := "-"
		if info.State != nil {
			hasState = true
			state = info.State.Status
			status = parseContainerStatusText(info)
		}

		image := info.Config.Image
		ports := parsePortSummary(info)

		containerInfos = append(containerInfos, containerInfo{
			Name:   name,
			Status: getStatusText(state, status, hasState),
			Image:  image,
			Ports:  ports,
			UserID: a.getUserID(name),
		})
	}

	strContainerResults := parseContainerInfos(containerInfos)

	for _, result := range strContainerResults {
		fmt.Println(result)
	}

	fmt.Println("[+] Runtime network status:")

	networkInfos := make([]networkInfo, 0)
	networkInfos = append(networkInfos, networkInfo{
		Name:   "NETWORK NAME",
		ID:     "NETWORK ID",
		Subnet: "SUBNET",
	})

	networks, err := a.dockerClient.NetworkList(a.context, network.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	for _, net := range networks {
		if strings.HasPrefix(net.Name, a.Config.UserService.Container.NetworkPrefix) ||
			net.Name == a.Config.ManagerService.Container.Network {
			info := networkInfo{
				Name:   net.Name,
				ID:     displayNetworkID(net.ID),
				Subnet: net.IPAM.Config[0].Subnet,
			}
			networkInfos = append(networkInfos, info)
		}
	}

	strNetResults := parseNetworkInfos(networkInfos)

	for _, result := range strNetResults {
		fmt.Println(result)
	}

	return nil
}

// getUserID extracts a display user identifier from a managed container name.
func (a *App) getUserID(name string) string {
	if strings.HasPrefix(name, a.Config.UserService.Container.NamePrefix) {
		return name[len(a.Config.UserService.Container.NamePrefix):]
	}
	if name == a.Config.AuthService.Container.Name {
		return "<AUTH SERVICE>"
	}
	if name == a.Config.ManagerService.Container.Name {
		return "<MANAGER SERVICE>"
	}
	return "-"
}

// displayNetworkID shortens long network IDs for status output.
func displayNetworkID(id string) string {
	if len(id) > 12 {
		return fmt.Sprintf("%s...", id[:12])
	}
	return id
}

// getStatusText combines Docker state and parsed status text for display.
func getStatusText(state, status string, hasState bool) string {
	if !hasState {
		return "-"
	} else {
		if state == status {
			return state
		} else {
			return fmt.Sprintf("%s(%s)", state, status)
		}
	}
}
