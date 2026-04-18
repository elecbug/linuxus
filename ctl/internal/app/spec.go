package app

import (
	"fmt"

	"github.com/elecbug/linuxus/src/ctl/internal/format"
	"github.com/elecbug/linuxus/src/ctl/internal/spec"
)

func (a *App) buildAuthRuntimeSpec() spec.RuntimeContainerSpec {
	return spec.RuntimeContainerSpec{
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
			"MANAGER_BASE_URL=" + fmt.Sprintf("http://%s:5959", a.Config.ManagerService.Container.Name),
			"MANAGER_TIMEOUT=" + a.Config.ManagerService.Session.Timeout,
			"MANAGER_SECRET=" + a.Config.ManagerService.Security.ManagerSecret,
		},
		Volumes: []string{
			fmt.Sprintf("%s:%s:rw", a.Config.AuthService.AuthListFile.HostPath, a.Config.AuthService.AuthListFile.ContainerPath),
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

func (a *App) buildManagerRuntimeSpec() (spec.RuntimeContainerSpec, error) {
	userCPUStr := fmt.Sprintf("%v", a.Config.UserService.Container.User.Limits.CPU)
	userNanoCPUs, err := format.StringToNanoCPUs(userCPUStr)
	if err != nil {
		return spec.RuntimeContainerSpec{}, fmt.Errorf("invalid user cpu limit: %w", err)
	}
	userMemBytes, err := format.StringToMemoryBytes(a.Config.UserService.Container.User.Limits.Memory)
	if err != nil {
		return spec.RuntimeContainerSpec{}, fmt.Errorf("invalid user memory limit: %w", err)
	}

	adminCPUStr := fmt.Sprintf("%v", a.Config.UserService.Container.Admin.Limits.CPU)
	adminNanoCPUs, err := format.StringToNanoCPUs(adminCPUStr)
	if err != nil {
		return spec.RuntimeContainerSpec{}, fmt.Errorf("invalid admin cpu limit: %w", err)
	}
	adminMemBytes, err := format.StringToMemoryBytes(a.Config.UserService.Container.Admin.Limits.Memory)
	if err != nil {
		return spec.RuntimeContainerSpec{}, fmt.Errorf("invalid admin memory limit: %w", err)
	}

	return spec.RuntimeContainerSpec{
		Image: a.managerImageName(),
		Name:  a.Config.ManagerService.Container.Name,
		Environment: []string{
			"TZ=" + a.Config.ManagerService.Container.Timezone,
			"USER_IMAGE=" + a.userImageName(),
			"USER_CONTAINER_NAME_PREFIX=" + a.Config.UserService.Container.NamePrefix,
			"NETWORK_PREFIX=" + a.Config.UserService.Container.NetworkPrefix,
			"BASE_IP=" + a.Config.UserService.Container.BaseIP,
			"AUTH_CONTAINER_NAME=" + a.Config.AuthService.Container.Name,
			"MANAGER_CONTAINER_NAME=" + a.Config.ManagerService.Container.Name,
			"ADMIN_USER_ID=" + a.Config.UserService.Container.Admin.UserID,

			"RUNTIME_USER=" + fmt.Sprintf("%d:%d", a.Config.UserService.Container.Runtime.UID, a.Config.UserService.Container.Runtime.GID),
			"CONTAINER_RUNTIME_USER=" + a.Config.UserService.Container.Runtime.User,
			"CONTAINER_HOSTNAME=" + a.Config.UserService.Container.Runtime.Hostname,
			"WORKING_DIR=" + "/home/" + a.Config.UserService.Container.Runtime.User,

			"HOST_HOMES_DIR=" + a.Config.Volumes.Host.Homes,
			"HOST_SHARE_DIR=" + a.Config.Volumes.Host.Share,
			"HOST_READONLY_DIR=" + a.Config.Volumes.Host.Readonly,
			"CONTAINER_SHARE_DIR=" + a.Config.Volumes.Container.Share,
			"CONTAINER_READONLY_DIR=" + a.Config.Volumes.Container.Readonly,

			"CONTAINER_USER_TIMEOUT=" + a.Config.ManagerService.User.Timeout,
			"MANAGER_WAIT_TIME=" + a.Config.ManagerService.Session.Timeout,
			"MANAGER_SECRET=" + a.Config.ManagerService.Security.ManagerSecret,
			"LISTEN_ADDR=:5959",

			"USER_NANO_CPUS=" + fmt.Sprintf("%d", userNanoCPUs),
			"USER_MEMORY_BYTES=" + fmt.Sprintf("%d", userMemBytes),
			"USER_PIDS_LIMIT=" + fmt.Sprintf("%d", a.Config.UserService.Container.User.Limits.PID),
			"USER_NOFILE_SOFT=" + fmt.Sprintf("%d", a.Config.UserService.Container.User.Limits.Ulimits.Nofile.Soft),
			"USER_NOFILE_HARD=" + fmt.Sprintf("%d", a.Config.UserService.Container.User.Limits.Ulimits.Nofile.Hard),
			"ADMIN_NANO_CPUS=" + fmt.Sprintf("%d", adminNanoCPUs),
			"ADMIN_MEMORY_BYTES=" + fmt.Sprintf("%d", adminMemBytes),
			"ADMIN_PIDS_LIMIT=" + fmt.Sprintf("%d", a.Config.UserService.Container.Admin.Limits.PID),
			"ADMIN_NOFILE_SOFT=" + fmt.Sprintf("%d", a.Config.UserService.Container.Admin.Limits.Ulimits.Nofile.Soft),
			"ADMIN_NOFILE_HARD=" + fmt.Sprintf("%d", a.Config.UserService.Container.Admin.Limits.Ulimits.Nofile.Hard),
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
