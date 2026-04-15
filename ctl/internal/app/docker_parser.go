package app

import (
	"fmt"
	"path/filepath"
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
	UserID string
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
	maxUserID := 0

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
		if len(info.UserID) > maxUserID {
			maxUserID = len(info.UserID)
		}
	}

	out := make([]string, len(infos))
	for i, info := range infos {
		out[i] = fmt.Sprintf("%-*s | %-*s | %-*s | %-*s | %-*s | %s",
			maxName, info.Name,
			maxUserID, info.UserID,
			maxState, info.State,
			maxStatus, info.Status,
			maxImage, info.Image,
			info.Ports,
		)
		if i == 0 {
			out[i] += "\n" + strings.Repeat("-", maxName) + "-|-" +
				strings.Repeat("-", maxUserID) + "-|-" + strings.Repeat("-", maxState) + "-|-" +
				strings.Repeat("-", maxStatus) + "-|-" + strings.Repeat("-", maxImage) + "-|-" +
				strings.Repeat("-", maxPorts)
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

func (a *App) buildAuthRuntimeSpec() RuntimeContainerSpec {
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
			"MANAGER_BASE_URL=" + fmt.Sprintf("http://%s:5959", a.Config.ManagerService.Container.Name),
			"MANAGER_TIMEOUT=" + a.Config.ManagerService.Session.Timeout,
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

func (a *App) buildManagerRuntimeSpec() RuntimeContainerSpec {
	return RuntimeContainerSpec{
		Image: a.managerImageName(),
		Name:  a.Config.ManagerService.Container.Name,
		Environment: []string{
			"TZ=" + a.Config.ManagerService.Container.Timezone,
			"USER_IMAGE=" + a.userImageName(),
			"USER_CONTAINER_NAME_PREFIX=" + a.Config.UserService.Container.NamePrefix,
			"NETWORK_PREFIX=" + a.Config.UserService.Container.NetworkPrefix,
			"BASE_IP=" + a.Config.UserService.Container.BaseIP,
			"AUTH_CONTAINER_NAME=" + a.Config.AuthService.Container.Name,
			"RUNTIME_USER=" + fmt.Sprintf("%d:%d", a.Config.UserService.Container.Runtime.UID, a.Config.UserService.Container.Runtime.GID),
			"WORKING_DIR=" + "/home/" + a.Config.UserService.Container.Runtime.User,
			"LISTEN_ADDR=:5959",
		},
		Volumes: []string{
			fmt.Sprintf("%s:/home/%s:rw", a.Config.Volumes.Host.Homes, a.Config.Volumes.Host.Homes),
			fmt.Sprintf("%s:%s:rw", a.Config.Volumes.Host.Share, a.Config.Volumes.Container.Share),
			fmt.Sprintf("%s:%s:rw", a.Config.Volumes.Host.Readonly, a.Config.Volumes.Container.Readonly),
			"/var/run/docker.sock:/var/run/docker.sock:rw",
		},
		Restart: "unless-stopped",
		Networks: []string{
			a.Config.ManagerService.Container.Network,
		},
	}
}

func (a *App) homeDirForUser(userID string) string {
	return filepath.Join(a.Config.Volumes.Host.Homes, userID)
}
