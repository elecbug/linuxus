package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

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

// ValidateConfig validates required config values before runtime operations.
func (a *App) ValidateConfig() error {
	if a.Config.AuthService.Container.ExternalPort <= 0 {
		return errors.New("auth_service.external_port must be a positive integer")
	}
	if a.Config.ManagerService.AdminID == "" {
		return errors.New("manager_service.admin_id must not be empty")
	}
	if a.Config.Volumes.Host.Volumes == "" {
		return errors.New("volumes.host.volumes must not be empty")
	}
	if a.Config.Volumes.Host.Homes == "" || a.Config.Volumes.Host.Share == "" || a.Config.Volumes.Host.Readonly == "" {
		return errors.New("volume host paths must not be empty")
	}
	if a.Config.Volumes.DiskLimit == "" {
		return errors.New("volumes.disk_limit must not be empty")
	}
	if a.Config.UserService.SourceDir == "" {
		return errors.New("user_service.source_dir must not be empty")
	}
	if a.Config.AuthService.SourceDir == "" {
		return errors.New("auth_service.source_dir must not be empty")
	}
	if a.Config.ManagerService.SourceDir == "" {
		return errors.New("manager_service.source_dir must not be empty")
	}
	if a.Config.ManagerService.Container.Name == "" {
		return errors.New("manager_service.container.name must not be empty")
	}
	if a.Config.ManagerService.Container.Network == "" {
		return errors.New("manager_service.container.network must not be empty")
	}
	if a.Config.ManagerService.AuthService.ConnectionTimeout == "" {
		return errors.New("manager_service.auth_service.connection_timeout must not be empty")
	}
	if _, err := time.ParseDuration(a.Config.ManagerService.AuthService.ConnectionTimeout); err != nil {
		return fmt.Errorf("manager_service.auth_service.connection_timeout is not a valid duration: %w", err)
	}
	if a.Config.ManagerService.Container.Subnet == "" {
		return errors.New("manager_service.container.subnet must not be empty")
	}
	if a.Config.UserService.Container.NamePrefix == "" {
		return errors.New("user_service.container.name_prefix must not be empty")
	}
	if a.Config.UserService.Container.NetworkNamePrefix == "" {
		return errors.New("user_service.container.network_name_prefix must not be empty")
	}
	if a.Config.UserService.Container.BaseSubnet16 == "" {
		return errors.New("user_service.container.base_subnet_16 must not be empty")
	}
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
