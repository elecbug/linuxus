package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// resolveStoragePath also resolves symlinked parents of paths not yet created.
func resolveStoragePath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err == nil {
		return resolved, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	if info, statErr := os.Lstat(absolute); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("storage symlink has a missing target: %s", absolute)
	}
	parent := filepath.Dir(absolute)
	if parent == absolute {
		return "", err
	}
	resolvedParent, err := resolveStoragePath(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(resolvedParent, filepath.Base(absolute)), nil
}

func containsPath(parent, path string) bool {
	rel, err := filepath.Rel(parent, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func validateVolumePaths(cfg *Config) error {
	paths := []string{cfg.Volumes.Host.Homes, cfg.Volumes.Host.Share, cfg.Volumes.Host.Readonly}
	// Images stay beside the configured mount paths even when the mount points
	// are symlinks. Validate their own resolved paths before preparing any disk.
	for _, mount := range []string{cfg.Volumes.Host.Share, cfg.Volumes.Host.Readonly} {
		if mount != "" {
			paths = append(paths, filepath.Clean(mount)+".img")
		}
	}
	files := make([]os.FileInfo, len(paths))
	for i, path := range paths {
		if path == "" {
			continue
		}
		resolved, err := resolveStoragePath(path)
		if err != nil {
			return fmt.Errorf("invalid volume path %q: %w", path, err)
		}
		if resolved == filepath.Dir(resolved) {
			return fmt.Errorf("volume path cannot be the filesystem root: %s", path)
		}
		paths[i] = resolved
		info, err := os.Stat(resolved)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("stat volume path %q: %w", path, err)
		}
		files[i] = info
	}
	for i, path := range paths {
		if path == "" {
			continue
		}
		for j := i + 1; j < len(paths); j++ {
			other := paths[j]
			sameFile := files[i] != nil && files[j] != nil && os.SameFile(files[i], files[j])
			if other != "" && (containsPath(path, other) || containsPath(other, path) || sameFile) {
				return fmt.Errorf("managed volume directories and images must not overlap: %s and %s", path, other)
			}
		}
	}
	if cfg.AuthService.Mounts.HostAuthListPath != "" {
		auth, err := resolveStoragePath(cfg.AuthService.Mounts.HostAuthListPath)
		if err != nil {
			return err
		}
		authInfo, err := os.Stat(auth)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		for i, path := range paths {
			sameFile := authInfo != nil && files[i] != nil && os.SameFile(authInfo, files[i])
			if path != "" && (containsPath(path, auth) || sameFile) {
				return fmt.Errorf("auth list must be outside managed disks and their images: %s", auth)
			}
		}
	}
	return nil
}

func validateContainerVolumePaths(cfg *Config) error {
	mounts := []struct{ name, target string }{
		{"user home", "/home/" + cfg.UserService.Runtime.LinuxUsername},
		{"VOLUMES_CONTAINER_SHARE", cfg.Volumes.Container.Share},
		{"VOLUMES_CONTAINER_READONLY", cfg.Volumes.Container.Readonly},
	}
	for i, mount := range mounts {
		if !path.IsAbs(mount.target) || path.Clean(mount.target) != mount.target || mount.target == "/" || strings.ContainsRune(mount.target, '\x00') {
			return fmt.Errorf("%s must be a clean absolute container path other than /", mount.name)
		}
		for _, other := range mounts[:i] {
			if mount.target == other.target || strings.HasPrefix(mount.target, other.target+"/") || strings.HasPrefix(other.target, mount.target+"/") {
				return fmt.Errorf("container volume paths must not overlap: %s and %s", other.name, mount.name)
			}
		}
	}
	return nil
}

// CanonicalStoragePath also resolves existing parents of not-yet-created paths.
func CanonicalStoragePath(path string) (string, error) { return resolveStoragePath(path) }
