package format

import (
	"fmt"
	"strings"
)

// DisplayNetworkID shortens network IDs for compact table output.
func DisplayNetworkID(id string) string {
	if len(id) > 12 {
		return fmt.Sprintf("%s...", id[:12])
	}
	return id
}

// DisplayStatusText formats state and status text consistently.
func DisplayStatusText(state, status string, hasState bool) string {
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

// DisplayUserName maps managed container names to display user identifiers.
func DisplayUserName(containerNamePrefix, authContainerName, managerContainerName, name string) string {
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
