package app

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/elecbug/linuxus/src/ctl/internal/format"
)

// PrepareUserDisks creates and mounts shared/admin/user disk images.
func (a *App) PrepareUserDisks() error {
	if err := a.systemAPI.MkdirAll(a.Config.Volumes.Host.Homes, 0755); err != nil {
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
	sizeStr := a.Config.Volumes.DiskLimit
	size, err := format.StringToBytes(sizeStr)
	if err != nil {
		return fmt.Errorf("invalid disk size for shared disk: %w", err)
	}
	if size <= 1024*1024 {
		return fmt.Errorf("volumes.disk_limit must be at least 1MB, got %d", size)
	}

	parentDir := filepath.Dir(path)
	name := filepath.Base(path)

	if err := a.systemAPI.MkdirAll(parentDir, 0755); err != nil {
		return err
	}

	img := filepath.Join(parentDir, name+".img")
	mountPoint := path

	if mounted, err := a.systemAPI.IsMountPoint(mountPoint); err != nil {
		return err
	} else if mounted {
		format.Log(format.INFO_PREFIX, "Already mounted: %s", mountPoint)
		return nil
	}

	if exists, err := a.systemAPI.Exists(img); err != nil {
		return fmt.Errorf("failed to stat image file %s: %w", img, err)
	} else if !exists {
		format.Log(format.RUN_PREFIX, "Creating shared disk for %s (%s)", mountPoint, sizeStr)

		if err := a.systemAPI.CreateEmptyFile(img, size); err != nil {
			return err
		}
		if err := a.systemAPI.FormatExt4(img); err != nil {
			return err
		}
	}

	if err := a.systemAPI.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point %s: %w", mountPoint, err)
	}

	loopdev, err := a.systemAPI.AttachLoopDevice(img)
	if err != nil {
		return err
	}

	mounted := false
	defer func() {
		if err == nil {
			return
		}
		if mounted {
			_ = a.systemAPI.Unmount(mountPoint)
		}
		_ = a.systemAPI.DetachLoopDevice(loopdev)
	}()

	if err = a.systemAPI.Mount(loopdev, mountPoint); err != nil {
		return err
	}
	mounted = true

	if err = a.systemAPI.Chown(
		mountPoint,
		a.Config.UserService.Runtime.UID,
		a.Config.UserService.Runtime.GID,
	); err != nil {
		return err
	}

	if err = a.systemAPI.Chmod(mountPoint, 0755); err != nil {
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
	if size <= 1024*1024 {
		userMode := "user"
		if isAdmin {
			userMode = "admin"
		}
		return fmt.Errorf("disk limit for %s must be at least 1MB, got %d", userMode, size)
	}

	img := filepath.Join(a.Config.Volumes.Host.Homes, userID+".img")
	mountPoint := filepath.Join(a.Config.Volumes.Host.Homes, userID)

	if mounted, err := a.systemAPI.IsMountPoint(mountPoint); err != nil {
		return err
	} else if mounted {
		format.Log(format.INFO_PREFIX, "Already mounted: %s", mountPoint)
		return nil
	}

	if exists, err := a.systemAPI.Exists(img); err != nil {
		return fmt.Errorf("failed to stat image file %s: %w", img, err)
	} else if !exists {
		format.Log(format.RUN_PREFIX, "Creating disk for %s (%s)", userID, sizeStr)

		if err := a.systemAPI.CreateEmptyFile(img, size); err != nil {
			return err
		}
		if err := a.systemAPI.FormatExt4(img); err != nil {
			return err
		}
	}

	if err := a.systemAPI.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point %s: %w", mountPoint, err)
	}

	loopdev, err := a.systemAPI.AttachLoopDevice(img)
	if err != nil {
		return err
	}

	mounted := false
	defer func() {
		if err == nil {
			return
		}
		if mounted {
			_ = a.systemAPI.Unmount(mountPoint)
		}
		_ = a.systemAPI.DetachLoopDevice(loopdev)
	}()

	if err = a.systemAPI.Mount(loopdev, mountPoint); err != nil {
		return err
	}
	mounted = true

	if err = a.systemAPI.Chown(
		mountPoint,
		a.Config.UserService.Runtime.UID,
		a.Config.UserService.Runtime.GID,
	); err != nil {
		return err
	}

	if err = a.systemAPI.Chmod(mountPoint, 0755); err != nil {
		return err
	}

	return nil
}

// listMountedDirsDeepestFirst returns mounted directories under root from deepest to shallowest.
func (a *App) listMountedDirsDeepestFirst(root string) ([]string, error) {
	var dirs []string

	exists, err := a.systemAPI.Exists(root)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if path == root {
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		mounted, err := a.systemAPI.IsMountPoint(path)
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

func (a *App) findLoopDevicesForImages(homesDir string) ([]string, error) {
	return a.systemAPI.FindLoopDevicesForImages(homesDir)
}

func (a *App) umountDisk(mountPoint string) error {
	mounted, err := a.systemAPI.IsMountPoint(mountPoint)
	if err != nil {
		return err
	}

	if mounted {
		format.Log(format.RUN_PREFIX, "Unmounting: %s", mountPoint)
		return a.systemAPI.Unmount(mountPoint)
	}

	return nil
}

func (a *App) detachLoopDevice(dev string) error {
	return a.systemAPI.DetachLoopDevice(dev)
}
