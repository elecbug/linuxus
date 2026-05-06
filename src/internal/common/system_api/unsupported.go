package system_api

import (
	"fmt"
	"os"
)

// UnsupportedSystemAPI implements SystemAPI for unsupported OSes, returning errors for all operations.
type UnsupportedSystemAPI struct {
	OS string
}

// MkdirAll returns an error indicating that mkdir is unsupported on this OS.
func (u UnsupportedSystemAPI) MkdirAll(path string, mode os.FileMode) error {
	return os.MkdirAll(path, mode)
}

// Chown returns an error indicating that chown is unsupported on this OS.
func (u UnsupportedSystemAPI) Chown(path string, uid, gid int) error {
	return fmt.Errorf("chown is unsupported on %s", u.OS)
}

// Chmod returns an error indicating that chmod is unsupported on this OS.
func (u UnsupportedSystemAPI) Chmod(path string, mode os.FileMode) error {
	return os.Chmod(path, mode)
}

// Exists checks if the specified path exists and is accessible.
func (u UnsupportedSystemAPI) Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// CreateEmptyFile creates an empty file at the specified path with the given size in bytes.
func (u UnsupportedSystemAPI) CreateEmptyFile(path string, sizeBytes int64) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %s: %w", path, err)
	}
	defer f.Close()

	if err := f.Truncate(sizeBytes); err != nil {
		return fmt.Errorf("failed to set file size: %s: %w", path, err)
	}
	return nil
}

// RemoveAll removes the specified path and all its contents.
func (u UnsupportedSystemAPI) RemoveAll(path string) error {
	if err := os.RemoveAll(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("failed to remove path: %s: %w", path, err)
	}
	return nil
}

// Remove removes the specified file.
func (u UnsupportedSystemAPI) Remove(path string) error {
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("failed to remove file: %s: %w", path, err)
	}
	return nil
}

// FormatExt4 returns an error indicating that formatting is unsupported on this OS.
func (u UnsupportedSystemAPI) FormatExt4(path string) error {
	return fmt.Errorf("formatting is unsupported on %s", u.OS)
}

// IsMountPoint returns an error indicating that mount point checking is unsupported on this OS.
func (u UnsupportedSystemAPI) IsMountPoint(path string) (bool, error) {
	return false, fmt.Errorf("mount point checking is unsupported on %s", u.OS)
}

// AttachLoopDevice returns an error indicating that loop device attachment is unsupported on this OS.
func (u UnsupportedSystemAPI) AttachLoopDevice(imagePath string) (string, error) {
	return "", fmt.Errorf("loop device attachment is unsupported on %s", u.OS)
}

// DetachLoopDevice returns an error indicating that loop device detachment is unsupported on this OS.
func (u UnsupportedSystemAPI) DetachLoopDevice(loopDevice string) error {
	return fmt.Errorf("loop device detachment is unsupported on %s", u.OS)
}

// Mount returns an error indicating that mounting is unsupported on this OS.
func (u UnsupportedSystemAPI) Mount(source string, target string) error {
	return fmt.Errorf("mounting is unsupported on %s", u.OS)
}

// Unmount returns an error indicating that unmount is unsupported on this OS.
func (u UnsupportedSystemAPI) Unmount(mountPoint string) error {
	return fmt.Errorf("unmount is unsupported on %s", u.OS)
}

// FindLoopDevicesForImages returns an error indicating that loop device listing is unsupported on this OS.
func (u UnsupportedSystemAPI) FindLoopDevicesForImages(dir string) ([]string, error) {
	return nil, fmt.Errorf("loop device listing is unsupported on %s", u.OS)
}
