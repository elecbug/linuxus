package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

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

func (a *App) ValidateConfig() error {
	if a.Config.AuthService.Container.ExternalPort <= 0 {
		return errors.New("auth_service.external_port must be a positive integer")
	}
	if a.Config.UserService.Container.Admin.UserID == "" {
		return errors.New("admin.user_id must not be empty")
	}
	if a.Config.Volumes.Host.Volumes == "" {
		return errors.New("volumes.host.volumes must not be empty")
	}
	if a.Config.Volumes.Host.Homes == "" || a.Config.Volumes.Host.Share == "" || a.Config.Volumes.Host.Readonly == "" {
		return errors.New("volume host paths must not be empty")
	}
	if a.Config.Volumes.DiskLimit <= 0 {
		return errors.New("volumes.disk_limit must be a positive integer")
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
	if a.Config.ManagerService.Session.Timeout == "" {
		return errors.New("manager_service.session.timeout must not be empty")
	}
	if _, err := time.ParseDuration(a.Config.ManagerService.Session.Timeout); err != nil {
		return fmt.Errorf("manager_service.session.timeout is not a valid duration: %w", err)
	}
	if a.Config.ManagerService.Container.Subnet == "" {
		return errors.New("manager_service.container.base_ip must not be empty")
	}
	if a.Config.UserService.Container.NamePrefix == "" {
		return errors.New("user_service.container.name_prefix must not be empty")
	}
	if a.Config.UserService.Container.NetworkPrefix == "" {
		return errors.New("user_service.container.network_prefix must not be empty")
	}
	if a.Config.UserService.Container.BaseIP == "" {
		return errors.New("user_service.container.base_ip must not be empty")
	}
	return nil
}

func (a *App) normalizeConfigPaths() {
	a.Config.UserService.SourceDir = a.absFromSource(a.Config.UserService.SourceDir)
	a.Config.AuthService.SourceDir = a.absFromSource(a.Config.AuthService.SourceDir)
	a.Config.ManagerService.SourceDir = a.absFromSource(a.Config.ManagerService.SourceDir)

	a.Config.AuthService.AuthListFile.HostPath = a.absFromSource(a.Config.AuthService.AuthListFile.HostPath)

	a.Config.Volumes.Host.Homes = a.absFromSource(a.Config.Volumes.Host.Homes)
	a.Config.Volumes.Host.Share = a.absFromSource(a.Config.Volumes.Host.Share)
	a.Config.Volumes.Host.Readonly = a.absFromSource(a.Config.Volumes.Host.Readonly)
	a.Config.Volumes.Host.Volumes = a.absFromSource(a.Config.Volumes.Host.Volumes)
}

func (a *App) absFromSource(path string) string {
	if path == "" {
		return path
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Clean(filepath.Join(a.sourceDir, path))
}
