package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultConfigFile = "/etc/linuxus/.env"
	DefaultStateDir   = "/var/lib/linuxus"
	ConfigFileEnv     = "LINUXUS_CONFIG"
)

// ResolveConfigFile deliberately does not search the working or executable
// directory: moving the CLI must not select a different installation's state.
func ResolveConfigFile() (string, error) {
	path, exists := os.LookupEnv(ConfigFileEnv)
	if !exists {
		return DefaultConfigFile, nil
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("%s must be an absolute configuration file path", ConfigFileEnv)
	}
	return filepath.Clean(path), nil
}

// WithConfigFile replaces existing entries so the spawned disk service reads
// the same configuration as the CLI that created it.
func WithConfigFile(environment []string, path string) []string {
	result := make([]string, 0, len(environment)+1)
	for _, entry := range environment {
		if !strings.HasPrefix(entry, ConfigFileEnv+"=") {
			result = append(result, entry)
		}
	}
	return append(result, ConfigFileEnv+"="+path)
}
