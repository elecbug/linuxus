package config

import (
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"testing"
)

func TestUsablePathDoesNotCreateParentDirectories(t *testing.T) {
	root := t.TempDir()
	if err := UsablePath(filepath.Join(root, "data", "AUTH_LIST")); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("validation created files: %v", entries)
	}
}

func TestRejectUnusableRuntimeSettings(t *testing.T) {
	data, err := os.ReadFile("../../../../cfg/config.yml")
	if err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*Config){
		"zero connection timeout":      func(c *Config) { c.ManagerService.AuthService.ConnectionTimeout = "0s" },
		"negative connection timeout":  func(c *Config) { c.ManagerService.AuthService.ConnectionTimeout = "-1s" },
		"negative cleanup timeout":     func(c *Config) { c.ManagerService.UserManagement.CleanupTimeout = "-1s" },
		"empty shared disk":            func(c *Config) { c.Volumes.DiskLimit = "0" },
		"too small user disk":          func(c *Config) { c.UserService.Limits.User.Disk = "1M" },
		"root home directory":          func(c *Config) { c.Volumes.Host.Homes = "/" },
		"overlapping disk directories": func(c *Config) { c.Volumes.Host.Share = filepath.Join(c.Volumes.Host.Homes, "share") },
		"credentials inside disk directory": func(c *Config) {
			c.AuthService.Mounts.HostAuthListPath = filepath.Join(c.Volumes.Host.Homes, "AUTH_LIST")
		},
	} {
		t.Run(name, func(t *testing.T) {
			var cfg Config
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			cfg.AuthService.Mounts.HostAuthListPath = filepath.Join(root, "data", "AUTH_LIST")
			cfg.Volumes.Host.Volumes = filepath.Join(root, "volumes")
			cfg.Volumes.Host.Homes = filepath.Join(root, "volumes", "homes")
			cfg.Volumes.Host.Share = filepath.Join(root, "volumes", "share")
			cfg.Volumes.Host.Readonly = filepath.Join(root, "volumes", "readonly")
			if err := ValidateConfig(&cfg); err != nil {
				t.Fatalf("invalid fixture: %v", err)
			}
			change(&cfg)
			if err := ValidateConfig(&cfg); err == nil {
				t.Fatal("accepted unusable setting")
			}
		})
	}
}

func TestVolumePathValidationResolvesSymlinks(t *testing.T) {
	root, storage := t.TempDir(), t.TempDir()
	alias := filepath.Join(root, "volumes")
	if err := os.Symlink(storage, alias); err != nil {
		t.Fatal(err)
	}
	var cfg Config
	cfg.Volumes.Host.Homes = filepath.Join(alias, "homes")
	cfg.Volumes.Host.Share = filepath.Join(storage, "share")
	cfg.Volumes.Host.Readonly = filepath.Join(storage, "readonly")
	cfg.AuthService.Mounts.HostAuthListPath = filepath.Join(root, "data", "AUTH_LIST")
	if err := validateVolumePaths(&cfg); err != nil {
		t.Fatal(err)
	}
	cfg.Volumes.Host.Share = filepath.Join(storage, "homes", "shared")
	if err := validateVolumePaths(&cfg); err == nil {
		t.Fatal("symlink hid overlapping volume paths")
	}
}

func TestAuthListCannotAliasSymlinkedShareImage(t *testing.T) {
	root, storage := t.TempDir(), t.TempDir()
	share := filepath.Join(root, "share")
	if err := os.Symlink(storage, share); err != nil {
		t.Fatal(err)
	}
	var cfg Config
	cfg.Volumes.Host.Share = share
	cfg.AuthService.Mounts.HostAuthListPath = share + ".img"
	if err := validateVolumePaths(&cfg); err == nil {
		t.Fatal("auth list would be deleted as a shared image")
	}
}
