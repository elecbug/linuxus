package config

import (
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
		"auth container in user namespace": func(c *Config) {
			c.AuthService.Container.Name = c.UserService.Container.NamePrefix + "auth"
		},
		"manager container in user namespace": func(c *Config) {
			c.ManagerService.Container.Name = c.UserService.Container.NamePrefix + "manager"
		},
		"manager network in user namespace": func(c *Config) {
			c.ManagerService.Container.Network = c.UserService.Container.NetworkNamePrefix + "manager"
		},
		"shared mount hides home": func(c *Config) {
			c.Volumes.Container.Share = "/home/" + c.UserService.Runtime.LinuxUsername
		},
		"readonly mount overlaps shared mount": func(c *Config) {
			c.Volumes.Container.Readonly = c.Volumes.Container.Share + "/readonly"
		},
		"unclean container mount": func(c *Config) {
			c.Volumes.Container.Share = "/home/user/../share"
		},
		"container root mount": func(c *Config) {
			c.Volumes.Container.Readonly = "/"
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg, err := ParseEnv([]byte(DefaultEnv))
			if err != nil {
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

func TestVolumeImagesCannotOverlapManagedStorage(t *testing.T) {
	for _, mode := range []string{"image-in-homes", "image-contains-mount", "symlinked-image-alias", "hardlinked-image-alias"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			var cfg Config
			cfg.Volumes.Host.Homes = filepath.Join(root, "homes")
			cfg.Volumes.Host.Share = filepath.Join(root, "share")
			cfg.Volumes.Host.Readonly = filepath.Join(root, "readonly")
			switch mode {
			case "image-in-homes":
				if err := os.Symlink(filepath.Join(cfg.Volumes.Host.Homes, "share.img"), cfg.Volumes.Host.Share+".img"); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(cfg.Volumes.Host.Homes, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(cfg.Volumes.Host.Homes, "share.img"), nil, 0600); err != nil {
					t.Fatal(err)
				}
			case "image-contains-mount":
				cfg.Volumes.Host.Readonly = filepath.Join(cfg.Volumes.Host.Share+".img", "readonly")
			case "symlinked-image-alias", "hardlinked-image-alias":
				if err := os.WriteFile(cfg.Volumes.Host.Share+".img", nil, 0600); err != nil {
					t.Fatal(err)
				}
				link := os.Symlink
				if mode == "hardlinked-image-alias" {
					link = os.Link
				}
				if err := link(cfg.Volumes.Host.Share+".img", cfg.Volumes.Host.Readonly+".img"); err != nil {
					t.Fatal(err)
				}
			}
			if err := validateVolumePaths(&cfg); err == nil {
				t.Fatal("accepted overlapping managed image and storage paths")
			}
		})
	}
}

func TestAuthListCannotHardlinkManagedImage(t *testing.T) {
	root := t.TempDir()
	var cfg Config
	cfg.Volumes.Host.Share = filepath.Join(root, "share")
	cfg.AuthService.Mounts.HostAuthListPath = filepath.Join(root, "AUTH_LIST")
	if err := os.WriteFile(cfg.Volumes.Host.Share+".img", nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(cfg.Volumes.Host.Share+".img", cfg.AuthService.Mounts.HostAuthListPath); err != nil {
		t.Fatal(err)
	}
	if err := validateVolumePaths(&cfg); err == nil {
		t.Fatal("auth list can refer to a managed image through a hard link")
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

func TestStoragePathTypes(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path             string
		directory, valid bool
	}{
		{root, true, true}, {file, false, true},
		{root, false, false}, {file, true, false},
		{filepath.Join(root, "missing"), true, true},
		{filepath.Join(root, "missing"), false, true},
		{filepath.Join(file, "invalid-parent"), true, false},
	} {
		if err := usableStoragePath(tc.path, tc.directory); (err == nil) != tc.valid {
			t.Fatalf("path=%s directory=%t error=%v", tc.path, tc.directory, err)
		}
	}
}
