package app

import (
	"fmt"
	"os"
	"path/filepath"

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
	if os.IsNotExist(err) {
		a.UserIDs = make(map[string]string)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to load user IDs from auth list: %w", err)
	}

	return nil
}

// normalizeConfigPaths resolves host paths relative to the deployment directory.
func (a *App) normalizeConfigPaths() {
	resolve := func(path string) string {
		if path == "" || filepath.IsAbs(path) {
			return path
		}
		return filepath.Join(a.runtimeRoot, path)
	}
	a.Config.AuthService.Mounts.HostAuthListPath = resolve(a.Config.AuthService.Mounts.HostAuthListPath)
	a.Config.Volumes.Host.Homes = resolve(a.Config.Volumes.Host.Homes)
	a.Config.Volumes.Host.Share = resolve(a.Config.Volumes.Host.Share)
	a.Config.Volumes.Host.Readonly = resolve(a.Config.Volumes.Host.Readonly)
	a.Config.Volumes.Host.Volumes = resolve(a.Config.Volumes.Host.Volumes)
}
