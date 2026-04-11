package app

import "fmt"

func (a *App) buildAuthService(adminSafe string) ComposeService {
	networks := make([]string, 0, len(a.SafeIDs)+1)
	for _, safeID := range a.SafeIDs {
		networks = append(networks, a.Config.UserService.NetworkPrefix+safeID)
	}
	networks = append(networks, a.Config.UserService.NetworkPrefix+adminSafe)

	return ComposeService{
		Build: &ComposeBuild{
			Context: a.Config.AuthService.SourceDir,
		},
		Container: a.Config.AuthService.ContainerName,
		Environment: []string{
			"TZ=" + a.Config.AuthService.Timezone,
			"AUTH_LIST=" + a.Config.AuthService.ListMountPath,
			"SESSION_SECRET=" + a.Config.AuthService.SessionSecret,
			"LOGIN_PATH=" + a.Config.URLPaths.Login,
			"LOGOUT_PATH=" + a.Config.URLPaths.Logout,
			"SERVICE_PATH=" + a.Config.URLPaths.Service,
			"TERMINAL_PATH=" + a.Config.URLPaths.Terminal,
			"USER_CONTAINER_NAME_PREFIX=" + a.Config.UserService.ContainerNamePrefix,
			"TRUSTED_PROXIES=" + a.Config.AuthService.TrustedProxies,
		},
		Volumes: []string{
			fmt.Sprintf("%s:%s:rw", a.Config.AuthService.ListFile, a.Config.AuthService.ListMountPath),
		},
		Ports: []string{
			fmt.Sprintf("%d:8080", a.Config.AuthService.ExternalPort),
		},
		Restart:  "unless-stopped",
		Networks: networks,
	}
}

func (a *App) buildUserService(userID, safeID string) ComposeService {
	return ComposeService{
		User: fmt.Sprintf("%d:%d", a.Config.ContainerRuntime.UID, a.Config.ContainerRuntime.GID),
		Build: &ComposeBuild{
			Context: a.Config.UserService.SourceDir,
			Args: []string{
				"CONTAINER_RUNTIME_USER=" + a.Config.ContainerRuntime.User,
			},
		},
		Container:  a.Config.UserService.ContainerNamePrefix + safeID,
		Hostname:   a.Config.ContainerRuntime.Hostname,
		WorkingDir: "/home/" + a.Config.ContainerRuntime.User,
		ReadOnly:   true,
		Tmpfs: []string{
			"/tmp:rw,noexec,nosuid,nodev,size=64m",
			"/run:rw,noexec,nosuid,nodev,size=16m",
			"/var/tmp:rw,noexec,nosuid,nodev,size=64m",
		},
		Environment: []string{
			"TZ=" + a.Config.ContainerRuntime.Timezone,
			"CONTAINER_RUNTIME_USER=" + a.Config.ContainerRuntime.User,
			"USER_ID=" + userID,
			"SHARED_DIR=" + a.Config.Volumes.Container.Share,
			"READONLY_DIR=" + a.Config.Volumes.Container.Readonly,
			"IS_ADMIN=false",
		},
		Volumes: []string{
			fmt.Sprintf("%s/%s:/home/%s:rw", a.Config.Volumes.Host.Homes, userID, a.Config.ContainerRuntime.User),
			fmt.Sprintf("%s:%s:rw", a.Config.Volumes.Host.Share, a.Config.Volumes.Container.Share),
			fmt.Sprintf("%s:%s:ro", a.Config.Volumes.Host.Readonly, a.Config.Volumes.Container.Readonly),
		},
		Restart:     "unless-stopped",
		SecurityOpt: []string{"no-new-privileges:true"},
		CapDrop:     []string{"ALL"},
		MemLimit:    a.Config.UserLimits.Memory,
		CPUs:        fmt.Sprintf("%v", a.Config.UserLimits.CPU),
		PidsLimit:   a.Config.UserLimits.PID,
		Ulimits: map[string]NofileLimit{
			"nofile": {
				Soft: a.Config.UserLimits.Ulimits.Nofile.Soft,
				Hard: a.Config.UserLimits.Ulimits.Nofile.Hard,
			},
		},
		Networks: []string{
			a.Config.UserService.NetworkPrefix + safeID,
		},
	}
}

func (a *App) buildAdminService(adminSafe string) ComposeService {
	return ComposeService{
		User: fmt.Sprintf("%d:%d", a.Config.ContainerRuntime.UID, a.Config.ContainerRuntime.GID),
		Build: &ComposeBuild{
			Context: a.Config.UserService.SourceDir,
			Args: []string{
				"CONTAINER_RUNTIME_USER=" + a.Config.ContainerRuntime.User,
			},
		},
		Container:  a.Config.UserService.ContainerNamePrefix + a.Config.Admin.UserID,
		Hostname:   a.Config.ContainerRuntime.Hostname,
		WorkingDir: "/home/" + a.Config.ContainerRuntime.User,
		ReadOnly:   true,
		Tmpfs: []string{
			"/tmp:rw,noexec,nosuid,nodev,size=64m",
			"/run:rw,noexec,nosuid,nodev,size=16m",
			"/var/tmp:rw,noexec,nosuid,nodev,size=64m",
		},
		Environment: []string{
			"TZ=" + a.Config.ContainerRuntime.Timezone,
			"CONTAINER_RUNTIME_USER=" + a.Config.ContainerRuntime.User,
			"USER_ID=" + a.Config.Admin.UserID,
			"SHARED_DIR=" + a.Config.Volumes.Container.Share,
			"READONLY_DIR=" + a.Config.Volumes.Container.Readonly,
			"IS_ADMIN=true",
		},
		Volumes: []string{
			fmt.Sprintf("%s/%s:/home/%s:rw", a.Config.Volumes.Host.Homes, a.Config.Admin.UserID, a.Config.ContainerRuntime.User),
			fmt.Sprintf("%s:%s:rw", a.Config.Volumes.Host.Share, a.Config.Volumes.Container.Share),
			fmt.Sprintf("%s:%s:rw", a.Config.Volumes.Host.Readonly, a.Config.Volumes.Container.Readonly),
		},
		Restart:     "unless-stopped",
		SecurityOpt: []string{"no-new-privileges:true"},
		CapDrop:     []string{"ALL"},
		MemLimit:    a.Config.AdminLimits.Memory,
		CPUs:        fmt.Sprintf("%v", a.Config.AdminLimits.CPU),
		PidsLimit:   a.Config.AdminLimits.PID,
		Ulimits: map[string]NofileLimit{
			"nofile": {
				Soft: a.Config.AdminLimits.Ulimits.Nofile.Soft,
				Hard: a.Config.AdminLimits.Ulimits.Nofile.Hard,
			},
		},
		Networks: []string{
			a.Config.UserService.NetworkPrefix + adminSafe,
		},
	}
}
