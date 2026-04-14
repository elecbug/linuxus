package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
)

type containerInfo struct {
	Name   string
	State  string
	Status string
	Image  string
	Ports  string
}

func parseNanoCPUs(v string) (int64, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return 0, err
	}
	if f < 0 {
		return 0, fmt.Errorf("must be non-negative")
	}
	return int64(f * 1_000_000_000), nil
}

func parseMemoryBytes(v string) (int64, error) {
	s := strings.TrimSpace(strings.ToLower(v))
	mult := int64(1)

	switch {
	case strings.HasSuffix(s, "g"):
		mult = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "g")
	case strings.HasSuffix(s, "gb"):
		mult = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "gb")
	case strings.HasSuffix(s, "m"):
		mult = 1024 * 1024
		s = strings.TrimSuffix(s, "m")
	case strings.HasSuffix(s, "mb"):
		mult = 1024 * 1024
		s = strings.TrimSuffix(s, "mb")
	case strings.HasSuffix(s, "k"):
		mult = 1024
		s = strings.TrimSuffix(s, "k")
	case strings.HasSuffix(s, "kb"):
		mult = 1024
		s = strings.TrimSuffix(s, "kb")
	case strings.HasSuffix(s, "b"):
		mult = 1
		s = strings.TrimSuffix(s, "b")
	}

	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, fmt.Errorf("must be non-negative")
	}
	return n * mult, nil
}

func parsePortBinding(s string) (nat.Port, nat.PortBinding, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return "", nat.PortBinding{}, fmt.Errorf("expected HOST:CONTAINER, got %q", s)
	}

	hostPort := strings.TrimSpace(parts[0])
	containerPart := strings.TrimSpace(parts[1])

	containerPort, err := nat.NewPort("tcp", containerPart)
	if err != nil {
		return "", nat.PortBinding{}, err
	}

	return containerPort, nat.PortBinding{
		HostIP:   "",
		HostPort: hostPort,
	}, nil
}

func parseContainerInfos(infos []containerInfo) []string {
	maxName := 0
	maxState := 0
	maxStatus := 0
	maxImage := 0
	maxPorts := 0

	for _, info := range infos {
		if len(info.Name) > maxName {
			maxName = len(info.Name)
		}
		if len(info.State) > maxState {
			maxState = len(info.State)
		}
		if len(info.Status) > maxStatus {
			maxStatus = len(info.Status)
		}
		if len(info.Image) > maxImage {
			maxImage = len(info.Image)
		}
		if len(info.Ports) > maxPorts {
			maxPorts = len(info.Ports)
		}
	}

	out := make([]string, len(infos))
	for i, info := range infos {
		out[i] = fmt.Sprintf("%-*s | %-*s | %-*s | %-*s | %s",
			maxName, info.Name,
			maxState, info.State,
			maxStatus, info.Status,
			maxImage, info.Image,
			info.Ports,
		)
		if i == 0 {
			out[i] += "\n" + strings.Repeat("-", maxName) + "-|-" + strings.Repeat("-", maxState) + "-|-" + strings.Repeat("-", maxStatus) + "-|-" + strings.Repeat("-", maxImage) + "-|-" + strings.Repeat("-", maxPorts)
		}
	}

	return out
}

func parseContainerStatusText(info container.InspectResponse) string {
	if info.State == nil {
		return "-"
	}

	status := info.State.Status

	if status == "exited" {
		return fmt.Sprintf("exited(%d)", info.State.ExitCode)
	}

	if info.State.OOMKilled {
		return "oom-killed"
	}

	return status
}

func parsePortSummary(info container.InspectResponse) string {
	if info.NetworkSettings == nil || len(info.NetworkSettings.Ports) == 0 {
		return "-"
	}

	first := true
	out := ""

	for containerPort, bindings := range info.NetworkSettings.Ports {
		if len(bindings) == 0 {
			if !first {
				out += ", "
			}
			out += string(containerPort)
			first = false
			continue
		}

		for _, b := range bindings {
			if !first {
				out += ", "
			}
			if b.HostIP != "" {
				out += fmt.Sprintf("%s:%s->%s", b.HostIP, b.HostPort, containerPort)
			} else {
				out += fmt.Sprintf("%s->%s", b.HostPort, containerPort)
			}
			first = false
		}
	}

	if out == "" {
		return "-"
	}
	return out
}

func parseSliceToTmpfsMap(items []string) map[string]string {
	if len(items) == 0 {
		return nil
	}

	out := make(map[string]string, len(items))
	for _, item := range items {
		parts := strings.SplitN(item, ":", 2)
		mountPoint := strings.TrimSpace(parts[0])
		if mountPoint == "" {
			continue
		}

		opts := ""
		if len(parts) == 2 {
			opts = strings.TrimSpace(parts[1])
		}
		out[mountPoint] = opts
	}
	return out
}

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
