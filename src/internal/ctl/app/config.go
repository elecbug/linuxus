package app

import (
	"fmt"
	"os"

	"github.com/elecbug/linuxus/src/internal/common/convert"
	"github.com/elecbug/linuxus/src/internal/common/user"
	"gopkg.in/yaml.v3"
)

// LoadConfig reads and parses the YAML configuration file into App.Config.
func (a *App) LoadConfig() error {
	data, err := os.ReadFile(a.configFile)
	if err != nil {
		return fmt.Errorf("config file not found: %s", a.configFile)
	}
	if err := yaml.Unmarshal(data, &a.Config); err != nil {
		return fmt.Errorf("failed to parse yaml config: %w", err)
	}

	a.normalizeConfigPaths()

	a.UserIDs, err = user.LoadUsers(a.Config.AuthService.Mounts.HostAuthListPath)
	if err != nil {
		return fmt.Errorf("failed to load user IDs from auth list: %w", err)
	}

	return nil
}

// normalizeConfigPaths resolves source-relative paths to absolute paths.
func (a *App) normalizeConfigPaths() {
	a.Config.AuthService.Mounts.HostAuthListPath = convert.PathToAbs(a.Config.AuthService.Mounts.HostAuthListPath)

	a.Config.Volumes.Host.Homes = convert.PathToAbs(a.Config.Volumes.Host.Homes)
	a.Config.Volumes.Host.Share = convert.PathToAbs(a.Config.Volumes.Host.Share)
	a.Config.Volumes.Host.Readonly = convert.PathToAbs(a.Config.Volumes.Host.Readonly)
	a.Config.Volumes.Host.Volumes = convert.PathToAbs(a.Config.Volumes.Host.Volumes)
}
