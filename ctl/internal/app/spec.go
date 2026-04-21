package app

import (
	"fmt"

	"github.com/elecbug/linuxus/src/ctl/internal/format"
	"github.com/elecbug/linuxus/src/ctl/internal/spec"
)

// buildAuthRuntimeSpec builds the auth service container runtime specification.
func (a *App) buildAuthRuntimeSpec() spec.RuntimeContainerSpec {
	return spec.RuntimeContainerSpec{
		Image: a.authImageName(),
		Name:  a.Config.AuthService.Container.Name,
		Environment: []string{
			"TZ=" + a.Config.AuthService.Runtime.Timezone,
			"AUTH_LIST=" + a.Config.AuthService.Mounts.ContainerAuthListPath,
			"SESSION_SECRET=" + a.Config.AuthService.Security.SessionSecret,
			"LOGIN_PATH=" + a.Config.AuthService.ServiceURL.Login,
			"LOGOUT_PATH=" + a.Config.AuthService.ServiceURL.Logout,
			"SERVICE_PATH=" + a.Config.AuthService.ServiceURL.Service,
			"TERMINAL_PATH=" + a.Config.AuthService.ServiceURL.Terminal,
			"USER_CONTAINER_NAME_PREFIX=" + a.Config.UserService.Container.NamePrefix,
			"TRUSTED_PROXIES=" + a.Config.AuthService.Security.TrustedProxies,
			"MANAGER_BASE_URL=" + fmt.Sprintf("http://%s:5959", a.Config.ManagerService.Container.Name),
			"MANAGER_TIMEOUT=" + a.Config.ManagerService.AuthService.ConnectionTimeout,
			"MANAGER_SECRET=" + a.Config.ManagerService.Security.ManagerSecret,
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
	}
}

// buildManagerRuntimeSpec builds the manager service container runtime specification.
func (a *App) buildManagerRuntimeSpec() (spec.RuntimeContainerSpec, error) {
	userCPUStr := fmt.Sprintf("%v", a.Config.UserService.Limits.User.CPU)
	userNanoCPUs, err := format.StringToNanoCPUs(userCPUStr)
	if err != nil {
		return spec.RuntimeContainerSpec{}, fmt.Errorf("invalid user cpu limit: %w", err)
	}
	userMemBytes, err := format.StringToBytes(a.Config.UserService.Limits.User.Memory)
	if err != nil {
		return spec.RuntimeContainerSpec{}, fmt.Errorf("invalid user memory limit: %w", err)
	}

	adminCPUStr := fmt.Sprintf("%v", a.Config.UserService.Limits.Admin.CPU)
	adminNanoCPUs, err := format.StringToNanoCPUs(adminCPUStr)
	if err != nil {
		return spec.RuntimeContainerSpec{}, fmt.Errorf("invalid admin cpu limit: %w", err)
	}
	adminMemBytes, err := format.StringToBytes(a.Config.UserService.Limits.Admin.Memory)
	if err != nil {
		return spec.RuntimeContainerSpec{}, fmt.Errorf("invalid admin memory limit: %w", err)
	}

	return spec.RuntimeContainerSpec{
		Image: a.managerImageName(),
		Name:  a.Config.ManagerService.Container.Name,
		Environment: []string{
			"USER_TZ=" + a.Config.UserService.Runtime.Timezone,
			"USER_IMAGE=" + a.userImageName(),
			"USER_CONTAINER_NAME_PREFIX=" + a.Config.UserService.Container.NamePrefix,
			"NETWORK_PREFIX=" + a.Config.UserService.Container.NetworkNamePrefix,
			"BASE_IP=" + a.Config.UserService.Container.BaseSubnet16,
			"AUTH_CONTAINER_NAME=" + a.Config.AuthService.Container.Name,
			"MANAGER_CONTAINER_NAME=" + a.Config.ManagerService.Container.Name,
			"ADMIN_USER_ID=" + a.Config.AuthService.AdminID,

			"RUNTIME_USER=" + fmt.Sprintf("%d:%d", a.Config.UserService.Runtime.UID, a.Config.UserService.Runtime.GID),
			"CONTAINER_RUNTIME_USER=" + a.Config.UserService.Runtime.LinuxUsername,
			"CONTAINER_HOSTNAME=" + a.Config.UserService.Runtime.LinuxHostname,
			"WORKING_DIR=" + "/home/" + a.Config.UserService.Runtime.LinuxUsername,

			"HOST_HOMES_DIR=" + a.Config.Volumes.Host.Homes,
			"HOST_SHARE_DIR=" + a.Config.Volumes.Host.Share,
			"HOST_READONLY_DIR=" + a.Config.Volumes.Host.Readonly,
			"CONTAINER_SHARE_DIR=" + a.Config.Volumes.Container.Share,
			"CONTAINER_READONLY_DIR=" + a.Config.Volumes.Container.Readonly,

			"CONTAINER_USER_TIMEOUT=" + a.Config.ManagerService.UserManagement.CleanupTimeout,
			"MANAGER_WAIT_TIME=" + a.Config.ManagerService.AuthService.ConnectionTimeout,
			"MANAGER_SECRET=" + a.Config.ManagerService.Security.ManagerSecret,
			"LISTEN_ADDR=:5959",

			"USER_NANO_CPUS=" + fmt.Sprintf("%d", userNanoCPUs),
			"USER_MEMORY_BYTES=" + fmt.Sprintf("%d", userMemBytes),
			"USER_PIDS_LIMIT=" + fmt.Sprintf("%d", a.Config.UserService.Limits.User.PID),
			"USER_NOFILE_SOFT=" + fmt.Sprintf("%d", a.Config.UserService.Limits.User.Ulimits.Nofile.Soft),
			"USER_NOFILE_HARD=" + fmt.Sprintf("%d", a.Config.UserService.Limits.User.Ulimits.Nofile.Hard),
			"ADMIN_NANO_CPUS=" + fmt.Sprintf("%d", adminNanoCPUs),
			"ADMIN_MEMORY_BYTES=" + fmt.Sprintf("%d", adminMemBytes),
			"ADMIN_PIDS_LIMIT=" + fmt.Sprintf("%d", a.Config.UserService.Limits.Admin.PID),
			"ADMIN_NOFILE_SOFT=" + fmt.Sprintf("%d", a.Config.UserService.Limits.Admin.Ulimits.Nofile.Soft),
			"ADMIN_NOFILE_HARD=" + fmt.Sprintf("%d", a.Config.UserService.Limits.Admin.Ulimits.Nofile.Hard),
		},
		Volumes: []string{
			fmt.Sprintf("%s:%s:rw", a.Config.Volumes.Host.Homes, a.Config.Volumes.Host.Homes),
			fmt.Sprintf("%s:%s:rw", a.Config.Volumes.Host.Share, a.Config.Volumes.Host.Share),
			fmt.Sprintf("%s:%s:rw", a.Config.Volumes.Host.Readonly, a.Config.Volumes.Host.Readonly),
			"/var/run/docker.sock:/var/run/docker.sock:rw",
		},
		Restart: "unless-stopped",
		Networks: []string{
			a.Config.ManagerService.Container.Network,
		},
	}, nil
}
