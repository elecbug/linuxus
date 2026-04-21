package app

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/elecbug/linuxus/src/ctl/internal/format"
)

// PrepareUserDisks creates and mounts shared/admin/user disk images.
func (a *App) PrepareUserDisks() error {
	if err := os.MkdirAll(a.Config.Volumes.Host.Homes, 0755); err != nil {
		return err
	}

	if err := a.createSharedDisk(a.Config.Volumes.Host.Share); err != nil {
		return err
	}
	if err := a.createSharedDisk(a.Config.Volumes.Host.Readonly); err != nil {
		return err
	}

	for _, userID := range a.UserIDs {
		if err := a.createUserDisk(userID, false); err != nil {
			return err
		}
	}
	if err := a.createUserDisk(a.Config.AuthService.AdminID, true); err != nil {
		return err
	}
	return nil
}

// createSharedDisk creates and mounts a shared loopback disk at the target path.
func (a *App) createSharedDisk(path string) error {
	size := a.Config.Volumes.DiskLimit
	if size <= 0 {
		return fmt.Errorf("volumes.disk_limit must be a positive integer, got %d", size)
	}

	parentDir := filepath.Dir(path)
	name := filepath.Base(path)

	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return err
	}

	img := filepath.Join(parentDir, name+".img")
	mountPoint := path

	if mounted, err := isMountPoint(mountPoint); err != nil {
		return err
	} else if mounted {
		fmt.Printf("[=] Already mounted: %s\n", mountPoint)
		return nil
	}

	if _, err := os.Stat(img); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to stat image file %s: %w", img, err)
		}
		fmt.Printf("[+] Creating shared disk for %s (%dMB)\n", mountPoint, size)
		if err := runCmd("sudo", "dd", "if=/dev/zero", "of="+img, "bs=1M", "count="+strconv.Itoa(size)); err != nil {
			return err
		}
		if err := runCmd("sudo", "mkfs.ext4", "-F", img); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point %s: %w", mountPoint, err)
	}

	loopdev, err := runCmdOutput("sudo", "losetup", "-f", "--show", img)
	if err != nil {
		return err
	}
	loopdev = strings.TrimSpace(loopdev)

	mounted := false
	defer func() {
		if err == nil {
			return
		}
		if mounted {
			_ = runCmd("sudo", "umount", mountPoint)
		}
		_ = runCmd("sudo", "losetup", "-d", loopdev)
	}()

	if err = runCmd("sudo", "mount", loopdev, mountPoint); err != nil {
		return err
	}
	mounted = true

	if err = runCmd("sudo", "chown",
		fmt.Sprintf("%d:%d", a.Config.UserService.Runtime.UID, a.Config.UserService.Runtime.GID),
		mountPoint); err != nil {
		return err
	}

	if err = runCmd("sudo", "chmod", "755", mountPoint); err != nil {
		return err
	}

	return nil
}

// createUserDisk creates and mounts a per-user loopback disk.
func (a *App) createUserDisk(userID string, isAdmin bool) error {
	sizeStr := a.Config.UserService.Limits.User.Disk
	if isAdmin {
		sizeStr = a.Config.UserService.Limits.Admin.Disk
	}

	size, err := format.StringToBytes(sizeStr)
	if err != nil {
		return fmt.Errorf("invalid disk size for %s: %w", userID, err)
	}

	if size <= 0 {
		userMode := "user"
		if isAdmin {
			userMode = "admin"
		}
		return fmt.Errorf("disk limit for %s must be a positive integer, got %d", userMode, size)
	}

	img := filepath.Join(a.Config.Volumes.Host.Homes, userID+".img")
	mountPoint := filepath.Join(a.Config.Volumes.Host.Homes, userID)

	if mounted, err := isMountPoint(mountPoint); err != nil {
		return err
	} else if mounted {
		fmt.Printf("[=] Already mounted: %s\n", mountPoint)
		return nil
	}

	if _, err := os.Stat(img); os.IsNotExist(err) {
		fmt.Printf("[+] Creating disk for %s (%dMB)\n", userID, size)
		if err := runCmd("sudo", "dd", "if=/dev/zero", "of="+img, "bs=1M", "count="+strconv.FormatInt(size, 10)); err != nil {
			return err
		}
		if err := runCmd("sudo", "mkfs.ext4", "-F", img); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point %s: %w", mountPoint, err)
	}

	loopdev, err := runCmdOutput("sudo", "losetup", "-f", "--show", img)
	if err != nil {
		return err
	}
	loopdev = strings.TrimSpace(loopdev)

	mounted := false
	defer func() {
		if err == nil {
			return
		}
		if mounted {
			_ = runCmd("sudo", "umount", mountPoint)
		}
		_ = runCmd("sudo", "losetup", "-d", loopdev)
	}()

	if err = runCmd("sudo", "mount", loopdev, mountPoint); err != nil {
		return err
	}
	mounted = true

	if err = runCmd("sudo", "chown",
		fmt.Sprintf("%d:%d", a.Config.UserService.Runtime.UID, a.Config.UserService.Runtime.GID),
		mountPoint); err != nil {
		return err
	}
	if err = runCmd("sudo", "chmod", "755", mountPoint); err != nil {
		return err
	}

	return nil
}

// listMountedDirsDeepestFirst returns mounted directories under root from deepest to shallowest.
func listMountedDirsDeepestFirst(root string) ([]string, error) {
	var dirs []string

	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil, nil
	}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path == root {
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		mounted, err := isMountPoint(path)
		if err != nil {
			return nil
		}
		if mounted {
			dirs = append(dirs, path)
			return fs.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(dirs, func(i, j int) bool {
		return len(dirs[i]) > len(dirs[j])
	})

	return dirs, nil
}

// isMountPoint reports whether a path is currently mounted.
func isMountPoint(path string) (bool, error) {
	cmd := exec.Command("mountpoint", "-q", path)
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("failed to check mountpoint %s: %w", path, err)
	}
	return false, fmt.Errorf("failed to check mountpoint %s: %w", path, err)
}

// findLoopDevicesForImages finds loop devices attached to image files in homesDir.
func findLoopDevicesForImages(homesDir string) ([]string, error) {
	entries, err := os.ReadDir(homesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read homes dir: %w", err)
	}

	// homes/*.img
	imageSet := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".img") {
			fullPath := filepath.Join(homesDir, name)
			absPath, err := filepath.Abs(fullPath)
			if err != nil {
				continue
			}
			imageSet[absPath] = struct{}{}
		}
	}

	if len(imageSet) == 0 {
		return nil, nil
	}

	out, err := runCmdOutput("sudo", "losetup", "-a")
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, nil
		}
		return nil, err
	}

	var loopDevs []string
	seen := make(map[string]struct{})

	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// e.g.: /dev/loop0: []: (/path/to/file.img)
		colonIdx := strings.Index(line, ":")
		lparenIdx := strings.LastIndex(line, "(")
		rparenIdx := strings.LastIndex(line, ")")

		if colonIdx <= 0 || lparenIdx < 0 || rparenIdx < 0 || rparenIdx <= lparenIdx {
			continue
		}

		dev := strings.TrimSpace(line[:colonIdx])
		backingFile := strings.TrimSpace(line[lparenIdx+1 : rparenIdx])

		absBacking, err := filepath.Abs(backingFile)
		if err != nil {
			continue
		}

		if _, ok := imageSet[absBacking]; ok {
			if _, exists := seen[dev]; !exists {
				seen[dev] = struct{}{}
				loopDevs = append(loopDevs, dev)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse losetup output: %w", err)
	}

	sort.Strings(loopDevs)
	return loopDevs, nil
}
