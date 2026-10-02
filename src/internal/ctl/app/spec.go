package app

import (
	"encoding/json"
	"fmt"

	"github.com/elecbug/linuxus/src/internal/ctl/spec"
)

// buildAuthRuntimeSpec builds the auth service container runtime specification.
func (a *App) buildAuthRuntimeSpec() (spec.RuntimeContainerSpec, error) {
	env, err := json.Marshal(a.Config)
	if err != nil {
		return spec.RuntimeContainerSpec{}, fmt.Errorf("failed to marshal config to JSON: %w", err)
	}

	return spec.RuntimeContainerSpec{
		Image: a.authImageName(),
		Name:  a.Config.AuthService.Container.Name,
		Environment: []string{
			"ENV=" + string(env),
			"TZ=" + a.Config.AuthService.Runtime.Timezone,
		},
		Volumes: []string{
			fmt.Sprintf("%s:%s:rw", a.Config.AuthService.Mounts.HostAuthListPath, a.Config.AuthService.Mounts.ContainerAuthListPath),
		},
		Ports: []string{
			fmt.Sprintf("%d:8080", a.Config.AuthService.Container.ExternalPort),
		},
		Restart: "unless-stopped",
		Networks: []string{
			a.Config.ManagerService.Container.Network,
		},
		Privileged: false,
	}, nil
}

// buildManagerRuntimeSpec builds the manager service container runtime specification.
func (a *App) buildManagerRuntimeSpec() (spec.RuntimeContainerSpec, error) {
	env, err := json.Marshal(a.Config)
	if err != nil {
		return spec.RuntimeContainerSpec{}, fmt.Errorf("failed to marshal config to JSON: %w", err)
	}

	return spec.RuntimeContainerSpec{
		Image: a.managerImageName(),
		Name:  a.Config.ManagerService.Container.Name,
		Environment: []string{
			"ENV=" + string(env),
			"USER_IMAGE=" + a.userImageName(),
		},
		Volumes: []string{
			"/var/run/docker.sock:/var/run/docker.sock:rw",
		},
		Restart: "unless-stopped",
		Networks: []string{
			a.Config.ManagerService.Container.Network,
		},
		Privileged: false,
	}, nil
}
