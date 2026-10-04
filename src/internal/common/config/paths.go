package config

import (
	"fmt"
	"os"
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
	}
	for i, path := range paths {
		if path == "" {
			continue
		}
		for _, other := range paths[i+1:] {
			if other != "" && (containsPath(path, other) || containsPath(other, path)) {
				return fmt.Errorf("home, shared and readonly volume paths must not overlap: %s and %s", path, other)
			}
		}
	}
	if cfg.AuthService.Mounts.HostAuthListPath != "" {
		auth, err := resolveStoragePath(cfg.AuthService.Mounts.HostAuthListPath)
		if err != nil {
			return err
		}
		for _, path := range paths {
			if path != "" && containsPath(path, auth) {
				return fmt.Errorf("auth list must be outside managed disks and their images: %s", auth)
			}
		}
		// A shared mount point may itself be a symlink; its image remains
		// beside the configured path rather than beside the resolved target.
		for _, mount := range []string{cfg.Volumes.Host.Share, cfg.Volumes.Host.Readonly} {
			if mount == "" {
				continue
			}
			image, err := resolveStoragePath(filepath.Clean(mount) + ".img")
			if err != nil {
				return err
			}
			if image == auth {
				return fmt.Errorf("auth list cannot be a managed disk image: %s", auth)
			}
		}
	}
	return nil
}

// CanonicalStoragePath also resolves existing parents of not-yet-created paths.
func CanonicalStoragePath(path string) (string, error) { return resolveStoragePath(path) }
