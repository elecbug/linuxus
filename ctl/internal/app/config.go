package app

import (
	"fmt"
	"os"
	"path/filepath"

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
	return nil
}

// normalizeConfigPaths resolves source-relative paths to absolute paths.
func (a *App) normalizeConfigPaths() {
	a.Config.UserService.SourceDir = a.absFromSource(a.Config.UserService.SourceDir)
	a.Config.AuthService.SourceDir = a.absFromSource(a.Config.AuthService.SourceDir)
	a.Config.ManagerService.SourceDir = a.absFromSource(a.Config.ManagerService.SourceDir)

	a.Config.AuthService.Mounts.HostAuthListPath = a.absFromSource(a.Config.AuthService.Mounts.HostAuthListPath)

	a.Config.Volumes.Host.Homes = a.absFromSource(a.Config.Volumes.Host.Homes)
	a.Config.Volumes.Host.Share = a.absFromSource(a.Config.Volumes.Host.Share)
	a.Config.Volumes.Host.Readonly = a.absFromSource(a.Config.Volumes.Host.Readonly)
	a.Config.Volumes.Host.Volumes = a.absFromSource(a.Config.Volumes.Host.Volumes)
}

// absFromSource resolves a path relative to the configured source directory.
func (a *App) absFromSource(path string) string {
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
