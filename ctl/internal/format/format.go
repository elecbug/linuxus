package format

import (
	"fmt"
	"strings"

	"github.com/elecbug/linuxus/src/ctl/internal/config"
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
func DisplayUserName(cfg config.Config, name string) string {
	if strings.HasPrefix(name, cfg.UserService.Container.NamePrefix) {
		return name[len(cfg.UserService.Container.NamePrefix):]
	}
	if name == cfg.AuthService.Container.Name {
		return "<AUTH SERVICE>"
	}
	if name == cfg.ManagerService.Container.Name {
		return "<MANAGER SERVICE>"
	}
	return "-"
}
