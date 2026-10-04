package config

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// DefaultEnv is copied from the project's settings and embedded at build time.
// Deployment files are never read during a build.
//
//go:embed default.env
var DefaultEnv string

// InitFile creates a private deployment configuration without overwriting an
// existing file or following a symlink. It needs neither Docker nor host mounts.
func InitFile(path string) (retErr error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create configuration directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("configuration already exists: %s; left unchanged", path)
		}
		return fmt.Errorf("create configuration %s: %w", path, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("close configuration: %w", err))
		}
		if retErr != nil {
			retErr = errors.Join(retErr, os.Remove(path))
		}
	}()
	if n, err := io.WriteString(file, DefaultEnv); err != nil {
		return fmt.Errorf("write configuration: %w", err)
	} else if n != len(DefaultEnv) {
		return io.ErrShortWrite
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync configuration: %w", err)
	}
	return nil
}
