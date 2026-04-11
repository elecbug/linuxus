package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func (a *App) LoadConfig() error {
	data, err := os.ReadFile(a.ConfigFile)
	if err != nil {
		return fmt.Errorf("config file not found: %s", a.ConfigFile)
	}
	if err := yaml.Unmarshal(data, &a.Config); err != nil {
		return fmt.Errorf("failed to parse yaml config: %w", err)
	}

	a.normalizeConfigPaths()
	return nil
}

func (a *App) ValidateConfig() error {
	if a.Config.AuthService.ExternalPort <= 0 {
		return errors.New("auth_service.external_port must be a positive integer")
	}
	if a.Config.Admin.UserID == "" {
		return errors.New("admin.user_id must not be empty")
	}
	if a.Config.Compose.OutputFile == "" {
		return errors.New("compose.output_file must not be empty")
	}
	if a.Config.Volumes.Host.Homes == "" || a.Config.Volumes.Host.Share == "" || a.Config.Volumes.Host.Readonly == "" {
		return errors.New("volume host paths must not be empty")
	}
	return nil
}

func (a *App) normalizeConfigPaths() {
	a.Config.UserService.SourceDir = a.absFromSource(a.Config.UserService.SourceDir)
	a.Config.AuthService.SourceDir = a.absFromSource(a.Config.AuthService.SourceDir)

	a.Config.AuthService.ListFile = a.absFromSource(a.Config.AuthService.ListFile)

	a.Config.Volumes.Host.Base = a.absFromSource(a.Config.Volumes.Host.Base)
	a.Config.Volumes.Host.Homes = a.absFromSource(a.Config.Volumes.Host.Homes)
	a.Config.Volumes.Host.Share = a.absFromSource(a.Config.Volumes.Host.Share)
	a.Config.Volumes.Host.Readonly = a.absFromSource(a.Config.Volumes.Host.Readonly)

	a.Config.Compose.OutputFile = a.absFromSource(a.Config.Compose.OutputFile)
}

func (a *App) absFromSource(path string) string {
	if path == "" {
		return path
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Clean(filepath.Join(a.SourceDir, path))
}
