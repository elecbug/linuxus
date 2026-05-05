package convert

import (
	"path/filepath"
)

// PathToAbs resolves a path relative to the configured source directory.
func PathToAbs(path string) string {
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
