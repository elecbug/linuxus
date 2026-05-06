package system_api

import (
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
func (LinuxSystemAPI) CreateEmptyFile(path string, sizeBytes int64) error {
	path = filepath.Clean(path)
	di := filepath.Dir(path)
	if err := os.MkdirAll(di, 0755); err != nil {
		return fmt.Errorf("failed to create parent directories for %s: %w", path, err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %s: %w", path, err)
	}
	defer f.Close()

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

// IsMountPoint checks if the specified path is a mount point by comparing device and inode numbers with its parent.
func (LinuxSystemAPI) IsMountPoint(path string) (bool, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}

	var st syscall.Stat_t
	var parent syscall.Stat_t

	if err := syscall.Stat(path, &st); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat failed: %s: %w", path, err)
	}

	parentPath := filepath.Dir(path)
	if parentPath == path {
		return true, nil
	}

	if err := syscall.Stat(parentPath, &parent); err != nil {
		return false, fmt.Errorf("parent stat failed: %s: %w", parentPath, err)
	}

	return st.Dev != parent.Dev || st.Ino == parent.Ino, nil
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

// FindLoopDevicesForImages returns a list of loop devices currently attached to image files under the specified directory.
func (LinuxSystemAPI) FindLoopDevicesForImages(dir string) ([]string, error) {
	out, err := exec.Command("losetup", "-a").Output()
	if err != nil {
		return nil, fmt.Errorf("losetup -a failed: %w", err)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	var devices []string

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		dev := strings.TrimSpace(parts[0])

		start := strings.LastIndex(line, "(")
		end := strings.LastIndex(line, ")")
		if start == -1 || end == -1 || end <= start+1 {
			continue
		}

		imgPath := line[start+1 : end]
		absImgPath, err := filepath.Abs(imgPath)
		if err != nil {
			continue
		}

		if strings.HasPrefix(absImgPath, absDir+string(os.PathSeparator)) {
			devices = append(devices, dev)
		}
	}

	return devices, nil
}
