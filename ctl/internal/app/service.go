package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/network"
	"github.com/elecbug/linuxus/src/ctl/internal/format"
	"github.com/elecbug/linuxus/src/ctl/internal/spec"
)

// ServiceUp builds images and starts all runtime-managed services.
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

// ServiceDown stops and removes all runtime-managed services.
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

// ServiceRestart recreates runtime-managed services.
func (a *App) ServiceRestart() error {
	fmt.Println("[+] Restarting runtime-managed containers...")
	if err := a.ServiceDown(); err != nil {
		return err
	}
	return a.ServiceUp()
}

// VolumeClean unmounts and removes managed volume data and loop devices.
func (a *App) VolumeClean() error {
	fmt.Println("[+] Cleaning volumes...")

	_ = a.ServiceDown()

	homeMounts, err := a.listMountedDirsDeepestFirst(a.Config.Volumes.Host.Homes)
	if err != nil {
		return err
	}
	for _, dir := range homeMounts {
		err = a.umountDisk(dir)
		if err != nil {
			fmt.Printf("[-] Failed to unmount home disk at %s: %v\n", dir, err)
			continue
		}
	}

	for _, mountPoint := range []string{a.Config.Volumes.Host.Share, a.Config.Volumes.Host.Readonly} {
		err = a.umountDisk(mountPoint)
		if err != nil {
			fmt.Printf("[-] Failed to unmount shared disk at %s: %v\n", mountPoint, err)
			continue
		}
	}

	homeDevs, err := a.findLoopDevicesForImages(a.Config.Volumes.Host.Homes)
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
		devs, err := a.findLoopDevicesForImages(filepath.Dir(mountPoint))
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
		err = a.detachLoopDevice(dev)
		if err != nil {
			fmt.Printf("[-] Failed to detach loop device %s: %v\n", dev, err)
			continue
		}
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

// ServicePS prints runtime status for managed containers and networks.
func (a *App) ServicePS() error {
	fmt.Println("[+] Runtime service status:")

	names, err := a.managedContainerNames()
	if err != nil {
		return err
	}

	containerInfos := make([]spec.ContainerInfo, 0, len(names)+1)
	containerInfos = append(containerInfos, spec.ContainerInfo{
		Name:   "CONTAINER NAME",
		Status: "STATE(STATUS)",
		Image:  "IMAGE",
		Ports:  "PORTS",
		Role:   "ROLE",
	})

	for _, name := range names {
		info, err := a.dockerClient.ContainerInspect(a.context, name)
		if err != nil {
			if errdefs.IsNotFound(err) {
				containerInfos = append(containerInfos, spec.ContainerInfo{
					Name:   name,
					Status: "not found",
					Image:  "-",
					Ports:  "-",
					Role:   "-",
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
			status = format.ContainerInspectToStatusText(info)
		}

		image := info.Config.Image
		ports := format.ContainerInspectToPortSummary(info)

		containerInfos = append(containerInfos, spec.ContainerInfo{
			Name:   name,
			Status: format.DisplayStatusText(state, status, hasState),
			Image:  image,
			Ports:  ports,
			Role:   format.DisplayUserName(a.Config, name),
		})
	}

	strContainerResults := format.ContainerInfosToStrings(containerInfos)

	for _, result := range strContainerResults {
		fmt.Println(result)
	}

	fmt.Println("[+] Runtime network status:")

	networkInfos := make([]spec.NetworkInfo, 0)
	networkInfos = append(networkInfos, spec.NetworkInfo{
		Name:   "NETWORK NAME",
		ID:     "NETWORK ID",
		Subnet: "SUBNET",
	})

	networks, err := a.dockerClient.NetworkList(a.context, network.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	for _, net := range networks {
		if strings.HasPrefix(net.Name, a.Config.UserService.Container.NetworkNamePrefix) ||
			net.Name == a.Config.ManagerService.Container.Network {
			info := spec.NetworkInfo{
				Name:   net.Name,
				ID:     format.DisplayNetworkID(net.ID),
				Subnet: net.IPAM.Config[0].Subnet,
			}
			networkInfos = append(networkInfos, info)
		}
	}

	strNetResults := format.NetworkInfosToStrings(networkInfos)

	for _, result := range strNetResults {
		fmt.Println(result)
	}

	return nil
}
