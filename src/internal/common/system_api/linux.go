package system_api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// LinuxSystemAPI implements SystemAPI for Linux using standard library and syscall.
type LinuxSystemAPI struct{}

// MkdirAll creates a directory and all necessary parents with the specified mode.
func (LinuxSystemAPI) MkdirAll(path string, mode os.FileMode) error {
	if err := os.MkdirAll(path, mode); err != nil {
		return fmt.Errorf("mkdir failed: %s: %w", path, err)
	}
	return nil
}

// Chown changes the ownership of the specified path to the given UID and GID.
func (LinuxSystemAPI) Chown(path string, uid, gid int) error {
	if err := os.Chown(path, uid, gid); err != nil {
		return fmt.Errorf("chown failed: %s: %w", path, err)
	}
	return nil
}

// Chmod changes the permissions of the specified path to the given mode.
func (LinuxSystemAPI) Chmod(path string, mode os.FileMode) error {
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("chmod failed: %s: %w", path, err)
	}
	return nil
}

// Exists checks if the specified path exists and is accessible.
func (LinuxSystemAPI) Exists(path string) (bool, error) {
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
func (LinuxSystemAPI) CreateEmptyFile(path string, sizeBytes int64) (retErr error) {
	path = filepath.Clean(path)
	di := filepath.Dir(path)
	if err := os.MkdirAll(di, 0755); err != nil {
		return fmt.Errorf("failed to create parent directories for %s: %w", path, err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to create file: %s: %w", path, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("close image %s: %w", path, err))
		}
		if retErr != nil {
			retErr = errors.Join(retErr, os.Remove(path))
		}
	}()

	if sizeBytes != 0 {
		if err := f.Truncate(sizeBytes); err != nil {
			return fmt.Errorf("failed to set file size: %s: %w", path, err)
		}
	}
	return nil
}

// RemoveAll removes the specified path and all its contents.
func (LinuxSystemAPI) RemoveAll(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to remove path: %s: %w", path, err)
	}
	return nil
}

// Remove removes the specified file.
func (LinuxSystemAPI) Remove(path string) error {
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("failed to remove file: %s: %w", path, err)
	}
	return nil
}

// FormatExt4 formats the specified file as an ext4 filesystem using mkfs.ext4.
func (LinuxSystemAPI) FormatExt4(path string) error {
	cmd := exec.Command("mkfs.ext4", "-F", "-q", path)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mkfs.ext4 failed: %s: %w", path, err)
	}

	return nil
}

// AttachLoopDevice attaches the specified image file to a free loop device and returns its path.
func (LinuxSystemAPI) AttachLoopDevice(imagePath string) (string, error) {
	cmd := exec.Command("losetup", "--find", "--show", imagePath)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf(
			"losetup attach failed: %s: %s: %w",
			imagePath,
			strings.TrimSpace(string(out)),
			err,
		)
	}

	loopdev := strings.TrimSpace(string(out))
	if loopdev == "" {
		return "", fmt.Errorf("losetup returned empty loop device for %s", imagePath)
	}

	return loopdev, nil
}

// DetachLoopDevice detaches the specified loop device using losetup -d.
func (LinuxSystemAPI) DetachLoopDevice(loopDevice string) error {
	if strings.TrimSpace(loopDevice) == "" {
		return nil
	}

	cmd := exec.Command("losetup", "-d", loopDevice)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("losetup detach failed: %s: %w", loopDevice, err)
	}

	return nil
}

// Mount mounts the source path to the target path using the syscall.Mount function with ext4 filesystem type.
func (LinuxSystemAPI) Mount(source string, target string) error {
	if err := syscall.Mount(source, target, "ext4", 0, ""); err != nil {
		return fmt.Errorf("mount failed: %s -> %s: %w", source, target, err)
	}

	return nil
}

// Unmount unmounts the specified target path using the syscall.Unmount function.
func (LinuxSystemAPI) Unmount(target string) error {
	if err := syscall.Unmount(target, 0); err != nil {
		return fmt.Errorf("unmount failed: %s: %w", target, err)
	}

	return nil
}

// FindLoopDevicesForImages accepts either an exact image path or a directory.
func (LinuxSystemAPI) FindLoopDevicesForImages(path string) ([]string, error) {
	out, err := exec.Command("losetup", "--json", "--output", "NAME,BACK-FILE").Output()
	if err != nil {
		return nil, fmt.Errorf("list loop devices: %w", err)
	}
	return loopDevicesForPath(out, path)
}

func loopDevicesForPath(data []byte, path string) ([]string, error) {
	absPath, err := canonicalLoopPath(path)
	if err != nil {
		return nil, err
	}
	var listing struct {
		Devices []struct {
			Name     string `json:"name"`
			BackFile string `json:"back-file"`
		} `json:"loopdevices"`
	}
	if err := json.Unmarshal(data, &listing); err != nil {
		return nil, fmt.Errorf("parse loop devices: %w", err)
	}
	var devices []string
	for _, dev := range listing.Devices {
		if dev.BackFile == "" || dev.Name == "" {
			continue
		}
		image, err := canonicalLoopPath(dev.BackFile)
		if err != nil {
			return nil, err
		}
		if image == absPath || strings.HasPrefix(image, absPath+string(os.PathSeparator)) {
			devices = append(devices, dev.Name)
		}
	}
	return devices, nil
}

// losetup reports canonical backing paths, including when the deployment uses
// a volumes symlink to storage outside the deployment directory.
func canonicalLoopPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if os.IsNotExist(err) {
		return abs, nil
	}
	return resolved, err
}
