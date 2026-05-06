package convert

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/elecbug/linuxus/src/internal/ctl/spec"
)

// ShortenNetworkID shortens network IDs for compact table output.
func ShortenNetworkID(id string) string {
	if len(id) > 12 {
		return fmt.Sprintf("%s...", id[:12])
	}
	return id
}

// FormatStatusText formats state and status text consistently.
func FormatStatusText(state, status string, hasState bool) string {
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

// FormatUserName maps managed container names to display user identifiers.
func FormatUserName(containerNamePrefix, authContainerName, managerContainerName, name string) string {
	if strings.HasPrefix(name, containerNamePrefix) {
		return fmt.Sprintf("<USER:%s>", name[len(containerNamePrefix):])
	}
	if name == authContainerName {
		return "<AUTH SERVICE>"
	}
	if name == managerContainerName {
		return "<MANAGER SERVICE>"
	}
	return "-"
}

// PathToAbs resolves a path relative to the configured source directory.
func PathToAbs(path string) string {
	if path == "" {
		return path
	}
	if filepath.IsAbs(path) {
		return path
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return filepath.Clean(absPath)
}

// NanoCPUsFromString converts a CPU value to Docker NanoCPUs.
func NanoCPUsFromString(v string) (int64, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return 0, err
	}
	if f < 0 {
		return 0, fmt.Errorf("must be non-negative")
	}
	return int64(f * 1_000_000_000), nil
}

// BytesFromString converts a memory string to bytes.
func BytesFromString(v string) (int64, error) {
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

// PortBindingFromString parses HOST:CONTAINER port mapping text.
func PortBindingFromString(s string) (nat.Port, nat.PortBinding, error) {
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

// ContainerInfosToStrings renders container info rows as aligned table lines.
func ContainerInfosToStrings(infos []spec.ContainerInfo) []string {
	maxName := 0
	maxStatus := 0
	maxImage := 0
	maxPorts := 0
	maxRole := 0

	for _, info := range infos {
		if len(info.Name) > maxName {
			maxName = len(info.Name)
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
		if len(info.Role) > maxRole {
			maxRole = len(info.Role)
		}
	}

	out := make([]string, len(infos))
	for i, info := range infos {
		out[i] = fmt.Sprintf("| %-*s | %-*s | %-*s | %-*s | %-*s |",
			maxName, info.Name,
			maxRole, info.Role,
			maxStatus, info.Status,
			maxImage, info.Image,
			maxPorts, info.Ports,
		)
		if i == 0 {
			out[i] += "\n|-" + strings.Repeat("-", maxName) + "-|-" +
				strings.Repeat("-", maxRole) + "-|-" + strings.Repeat("-", maxStatus) + "-|-" +
				strings.Repeat("-", maxImage) + "-|-" + strings.Repeat("-", maxPorts) + "-|"
		}
	}

	return out
}

// NetworkInfosToStrings renders network info rows as aligned table lines.
func NetworkInfosToStrings(infos []spec.NetworkInfo) []string {
	maxName := 0
	maxID := 0
	maxSubnet := 0

	for _, info := range infos {
		if len(info.Name) > maxName {
			maxName = len(info.Name)
		}
		if len(info.ID) > maxID {
			maxID = len(info.ID)
		}
		if len(info.Subnet) > maxSubnet {
			maxSubnet = len(info.Subnet)
		}
	}

	out := make([]string, len(infos))
	for i, info := range infos {
		out[i] = fmt.Sprintf("| %-*s | %-*s | %-*s |",
			maxName, info.Name,
			maxID, info.ID,
			maxSubnet, info.Subnet,
		)
		if i == 0 {
			out[i] += "\n|-" + strings.Repeat("-", maxName) + "-|-" +
				strings.Repeat("-", maxID) + "-|-" + strings.Repeat("-", maxSubnet) + "-|"
		}
	}

	return out
}

// ContainerInspectToStatusText derives a concise status from Docker inspect data.
func ContainerInspectToStatusText(info container.InspectResponse) string {
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

// ContainerInspectToPortSummary converts exposed ports to a display string.
func ContainerInspectToPortSummary(info container.InspectResponse) string {
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

// TmpfsMap converts tmpfs mount strings to Docker tmpfs map format.
func TmpfsMap(items []string) map[string]string {
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
