package app

import (
	"fmt"
	"strings"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/network"
	"github.com/elecbug/linuxus/src/internal/common/convert"
	"github.com/elecbug/linuxus/src/internal/ctl/format"
	"github.com/elecbug/linuxus/src/internal/ctl/spec"
)

// showContainerInfos retrieves and displays information about the runtime-managed containers.
func (a *App) showContainerInfos() error {
	format.Log(format.DETAIL_PREFIX, "Runtime service status:")

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
			Status: convert.FormatStatusText(state, status, hasState),
			Image:  image,
			Ports:  ports,
			Role: convert.FormatUserName(
				a.Config.UserService.Container.NamePrefix,
				a.Config.AuthService.Container.Name,
				a.Config.ManagerService.Container.Name,
				name,
			),
		})
	}

	strContainerResults := format.ContainerInfosToStrings(containerInfos)

	for _, result := range strContainerResults {
		fmt.Println(result)
	}

	return nil
}

// showNetworkInfos retrieves and displays information about the runtime networks used by the services.
func (a *App) showNetworkInfos() error {
	format.Log(format.DETAIL_PREFIX, "Runtime network status:")

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
				ID:     convert.ShortenNetworkID(net.ID),
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
