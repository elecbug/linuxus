package app

import "fmt"

func (a *App) buildAuthRuntimeSpec(adminSafe string) RuntimeContainerSpec {
	networks := make([]string, 0, len(a.SafeIDs)+1)
	for _, safeID := range a.SafeIDs {
		networks = append(networks, a.Config.UserService.Container.NetworkPrefix+safeID)
	}
	networks = append(networks, a.Config.UserService.Container.NetworkPrefix+adminSafe)

	return RuntimeContainerSpec{
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
		},
		Volumes: []string{
			fmt.Sprintf("%s:%s:rw", a.Config.AuthService.AuthListFile.HostPath, a.Config.AuthService.AuthListFile.ContainerPath),
		},
		Ports: []string{
			fmt.Sprintf("%d:8080", a.Config.AuthService.Container.ExternalPort),
		},
		Restart:  "unless-stopped",
		Networks: networks,
	}
}

func (a *App) buildUserRuntimeSpec(userID, safeID string) RuntimeContainerSpec {
	return RuntimeContainerSpec{
		Image:      a.userImageName(),
		Name:       a.Config.UserService.Container.NamePrefix + safeID,
		User:       fmt.Sprintf("%d:%d", a.Config.UserService.Container.Runtime.UID, a.Config.UserService.Container.Runtime.GID),
		Hostname:   a.Config.UserService.Container.Runtime.Hostname,
		WorkingDir: "/home/" + a.Config.UserService.Container.Runtime.User,
		ReadOnly:   true,
		Tmpfs: []string{
			"/tmp:rw,noexec,nosuid,nodev,size=64m",
			"/run:rw,noexec,nosuid,nodev,size=16m",
			"/var/tmp:rw,noexec,nosuid,nodev,size=64m",
		},
		Environment: []string{
			"TZ=" + a.Config.UserService.Container.Runtime.Timezone,
			"CONTAINER_RUNTIME_USER=" + a.Config.UserService.Container.Runtime.User,
			"USER_ID=" + userID,
			"SHARED_DIR=" + a.Config.Volumes.Container.Share,
			"READONLY_DIR=" + a.Config.Volumes.Container.Readonly,
			"IS_ADMIN=false",
		},
		Volumes: []string{
			fmt.Sprintf("%s:/home/%s:rw", a.homeDirForUser(userID), a.Config.UserService.Container.Runtime.User),
			fmt.Sprintf("%s:%s:rw", a.Config.Volumes.Host.Share, a.Config.Volumes.Container.Share),
			fmt.Sprintf("%s:%s:ro", a.Config.Volumes.Host.Readonly, a.Config.Volumes.Container.Readonly),
		},
		Restart:     "unless-stopped",
		SecurityOpt: []string{"no-new-privileges:true"},
		CapDrop:     []string{"ALL"},
		Limits: ContainerLimits{
			Memory:     a.Config.UserService.Container.User.Limits.Memory,
			CPUs:       fmt.Sprintf("%v", a.Config.UserService.Container.User.Limits.CPU),
			Pids:       a.Config.UserService.Container.User.Limits.PID,
			NofileSoft: a.Config.UserService.Container.User.Limits.Ulimits.Nofile.Soft,
			NofileHard: a.Config.UserService.Container.User.Limits.Ulimits.Nofile.Hard,
		},
		Networks: []string{
			a.Config.UserService.Container.NetworkPrefix + safeID,
		},
	}
}

func (a *App) buildAdminRuntimeSpec(adminSafe string) RuntimeContainerSpec {
	return RuntimeContainerSpec{
		Image:      a.userImageName(),
		Name:       a.Config.UserService.Container.NamePrefix + a.Config.UserService.Container.Admin.UserID,
		User:       fmt.Sprintf("%d:%d", a.Config.UserService.Container.Runtime.UID, a.Config.UserService.Container.Runtime.GID),
		Hostname:   a.Config.UserService.Container.Runtime.Hostname,
		WorkingDir: "/home/" + a.Config.UserService.Container.Runtime.User,
		ReadOnly:   true,
		Tmpfs: []string{
			"/tmp:rw,noexec,nosuid,nodev,size=64m",
			"/run:rw,noexec,nosuid,nodev,size=16m",
			"/var/tmp:rw,noexec,nosuid,nodev,size=64m",
		},
		Environment: []string{
			"TZ=" + a.Config.UserService.Container.Runtime.Timezone,
			"CONTAINER_RUNTIME_USER=" + a.Config.UserService.Container.Runtime.User,
			"USER_ID=" + a.Config.UserService.Container.Admin.UserID,
			"SHARED_DIR=" + a.Config.Volumes.Container.Share,
			"READONLY_DIR=" + a.Config.Volumes.Container.Readonly,
			"IS_ADMIN=true",
		},
		Volumes: []string{
			fmt.Sprintf("%s:/home/%s:rw", a.homeDirForUser(a.Config.UserService.Container.Admin.UserID), a.Config.UserService.Container.Runtime.User),
			fmt.Sprintf("%s:%s:rw", a.Config.Volumes.Host.Share, a.Config.Volumes.Container.Share),
			fmt.Sprintf("%s:%s:rw", a.Config.Volumes.Host.Readonly, a.Config.Volumes.Container.Readonly),
		},
		Restart:     "unless-stopped",
		SecurityOpt: []string{"no-new-privileges:true"},
		CapDrop:     []string{"ALL"},
		Limits: ContainerLimits{
			Memory:     a.Config.UserService.Container.Admin.Limits.Memory,
			CPUs:       fmt.Sprintf("%v", a.Config.UserService.Container.Admin.Limits.CPU),
			Pids:       a.Config.UserService.Container.Admin.Limits.PID,
			NofileSoft: a.Config.UserService.Container.Admin.Limits.Ulimits.Nofile.Soft,
			NofileHard: a.Config.UserService.Container.Admin.Limits.Ulimits.Nofile.Hard,
		},
		Networks: []string{
			a.Config.UserService.Container.NetworkPrefix + adminSafe,
		},
	}
}

func (a *App) buildRuntimeNetworks() ([]RuntimeNetworkSpec, error) {
	networks := make([]RuntimeNetworkSpec, 0, len(a.SafeIDs)+1)
	seq := 0

	for _, safeID := range a.SafeIDs {
		subnet, err := getIP(a.Config.UserService.Container.BaseIP, seq)
		if err != nil {
			return nil, err
		}
		networks = append(networks, RuntimeNetworkSpec{
			Name:   a.Config.UserService.Container.NetworkPrefix + safeID,
			Subnet: subnet,
		})
		seq++
	}

	adminSafe := sanitizeName(a.Config.UserService.Container.Admin.UserID)
	subnet, err := getIP(a.Config.UserService.Container.BaseIP, seq)
	if err != nil {
		return nil, err
	}
	networks = append(networks, RuntimeNetworkSpec{
		Name:   a.Config.UserService.Container.NetworkPrefix + adminSafe,
		Subnet: subnet,
	})

	return networks, nil
}
