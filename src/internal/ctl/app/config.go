package app

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/elecbug/linuxus/src/internal/common/config"
	"github.com/elecbug/linuxus/src/internal/common/user"
	"gopkg.in/yaml.v3"
)

// LoadConfig reads and parses the YAML configuration file into App.Config.
func (a *App) LoadConfig() error {
	data, err := os.ReadFile(a.configFile)
	if err != nil {
		return fmt.Errorf("read config %s: %w", a.configFile, err)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var parsed config.Config
	if err := decoder.Decode(&parsed); err != nil {
		return fmt.Errorf("failed to parse yaml config: %w", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return fmt.Errorf("failed to parse trailing yaml: %w", err)
		}
		return fmt.Errorf("config must contain exactly one YAML document")
	}
	a.Config = parsed

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
		if path == "" {
			return ""
		}
		if filepath.IsAbs(path) {
			return filepath.Clean(path)
		}
		return filepath.Join(a.runtimeRoot, path)
	}
	a.Config.AuthService.Mounts.HostAuthListPath = resolve(a.Config.AuthService.Mounts.HostAuthListPath)
	a.Config.Volumes.Host.Homes = resolve(a.Config.Volumes.Host.Homes)
	a.Config.Volumes.Host.Share = resolve(a.Config.Volumes.Host.Share)
	a.Config.Volumes.Host.Readonly = resolve(a.Config.Volumes.Host.Readonly)
	a.Config.Volumes.Host.Volumes = resolve(a.Config.Volumes.Host.Volumes)
}
