package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/elecbug/linuxus/src/internal/common/config"
	"github.com/elecbug/linuxus/src/internal/common/user"
)

// loadSettings reads settings independently of Docker and the account database.
func (a *App) loadSettings() error {
	// Resolve config symlinks before interpreting relative storage paths so
	// both a symlink and its target select the same persistent state.
	configFile, err := filepath.EvalSymlinks(a.configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config %s does not exist; run sudo linuxusctl init: %w", a.configFile, err)
		}
		return fmt.Errorf("resolve config %s: %w", a.configFile, err)
	}
	a.configFile, err = filepath.Abs(configFile)
	if err != nil {
		return err
	}
	info, err := os.Stat(a.configFile)
	if err != nil {
		return fmt.Errorf("stat config %s: %w", a.configFile, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("config %s must be a regular file", a.configFile)
	}
	data, err := os.ReadFile(a.configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config %s does not exist; run sudo linuxusctl init: %w", a.configFile, err)
		}
		return fmt.Errorf("read config %s: %w", a.configFile, err)
	}
	parsed, err := config.ParseEnv(data)
	if err != nil {
		return fmt.Errorf("parse config %s: %w", a.configFile, err)
	}
	a.Config = parsed

	a.normalizeConfigPaths()
	return nil
}

// LoadConfig also loads credentials for commands that operate on accounts.
func (a *App) LoadConfig() error {
	if err := a.loadSettings(); err != nil {
		return err
	}
	users, err := user.LoadUsers(a.Config.AuthService.Mounts.HostAuthListPath)
	if os.IsNotExist(err) {
		a.UserIDs = make(map[string]string)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to load user IDs from auth list: %w", err)
	}

	a.UserIDs = users
	return nil
}

// normalizeConfigPaths resolves host paths relative to the selected configuration directory.
func (a *App) normalizeConfigPaths() {
	resolve := func(path string) string {
		if path == "" {
			return ""
		}
		if filepath.IsAbs(path) {
			return filepath.Clean(path)
		}
		return filepath.Join(filepath.Dir(a.configFile), path)
	}
	a.Config.AuthService.Mounts.HostAuthListPath = resolve(a.Config.AuthService.Mounts.HostAuthListPath)
	a.Config.Volumes.Host.Homes = resolve(a.Config.Volumes.Host.Homes)
	a.Config.Volumes.Host.Share = resolve(a.Config.Volumes.Host.Share)
	a.Config.Volumes.Host.Readonly = resolve(a.Config.Volumes.Host.Readonly)
	a.Config.Volumes.Host.Volumes = resolve(a.Config.Volumes.Host.Volumes)
}
