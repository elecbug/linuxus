package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/elecbug/linuxus/src/internal/common/convert"
	"github.com/elecbug/linuxus/src/internal/common/ruleset"
	"github.com/elecbug/linuxus/src/internal/common/user"
	"github.com/elecbug/linuxus/src/internal/ctl/log"
)

// cleanVolumesAll unmounts and removes all user and shared disks after confirming with the user.
func (a *App) cleanVolumesAll() error {
	yes, err := log.Input("Are you sure you want to clean volumes for ALL users? This action cannot be undone. (yes/no): ")
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}

	if strings.ToLower(yes) != "yes" {
		log.Log(log.INFO_PREFIX, "Volume clean cancelled.")
		return nil
	}

	log.Log(log.RUN_PREFIX, "Cleaning volumes for all users...")
	log.Log(log.INFO_PREFIX, "Stopping and removing all managed containers and networks...")

	if err := a.removeManagedContainers(); err != nil {
		return err
	}
	if err := a.stopDiskService(); err != nil {
		return err
	}
	if err := a.removeManagedNetworks(); err != nil {
		return err
	}
	return a.withDiskLock(a.cleanVolumesAllUnlocked)
}

func (a *App) cleanVolumesAllUnlocked() error {
	homeMounts, err := a.listMountedDirsDeepestFirst(a.Config.Volumes.Host.Homes)
	if err != nil {
		return err
	}
	for _, dir := range homeMounts {
		err = a.umountDisk(dir)
		if err != nil {
			return fmt.Errorf("unmount home disk %s: %w", dir, err)
		}
	}

	for _, mountPoint := range []string{a.Config.Volumes.Host.Share, a.Config.Volumes.Host.Readonly} {
		err = a.unmountDiskTree(mountPoint)
		if err != nil {
			return fmt.Errorf("unmount shared disk %s: %w", mountPoint, err)
		}
	}

	homeDevs, err := a.findLoopDevicesForImages(a.Config.Volumes.Host.Homes)
	if err != nil {
		return err
	}

	seen := make(map[string]struct{})
	var loopDevs []string
	for _, dev := range homeDevs {
		if _, exists := seen[dev]; !exists {
			seen[dev] = struct{}{}
			loopDevs = append(loopDevs, dev)
		}
	}
	for _, mountPoint := range []string{a.Config.Volumes.Host.Share, a.Config.Volumes.Host.Readonly} {
		devs, err := a.findLoopDevicesForImages(mountPoint + ".img")
		if err != nil {
			return err
		}
		for _, dev := range devs {
			if _, exists := seen[dev]; !exists {
				seen[dev] = struct{}{}
				loopDevs = append(loopDevs, dev)
			}
		}
	}

	for _, dev := range loopDevs {
		log.Log(log.DETAIL_PREFIX, "Detaching loop device: %s", dev)
		err = a.detachLoopDevice(dev)
		if err != nil {
			return fmt.Errorf("detach loop device %s: %w", dev, err)
		}
	}

	if err := a.systemAPI.RemoveAll(a.Config.Volumes.Host.Homes); err != nil {
		return fmt.Errorf("failed to remove homes dir: %w", err)
	}
	if err := a.systemAPI.RemoveAll(a.Config.Volumes.Host.Share); err != nil {
		return fmt.Errorf("failed to remove share dir: %w", err)
	}
	if err := a.systemAPI.RemoveAll(a.Config.Volumes.Host.Readonly); err != nil {
		return fmt.Errorf("failed to remove readonly dir: %w", err)
	}
	for _, mountPoint := range []string{a.Config.Volumes.Host.Share, a.Config.Volumes.Host.Readonly} {
		if err := a.systemAPI.Remove(mountPoint + ".img"); err != nil {
			return fmt.Errorf("remove shared disk image: %w", err)
		}
	}

	log.Log(log.DETAIL_PREFIX, "Volume clean completed.")
	return nil
}

// cleanVolumeUser unmounts and removes the specified user's disk and home directory.
func (a *App) cleanVolumeUser(userID string) error {
	if !ruleset.AllowedUserID(userID) {
		return fmt.Errorf("invalid user ID: %q", userID)
	}
	return a.withDiskLock(func() error { return a.cleanVolumeUserUnlocked(userID) })
}

func (a *App) cleanVolumeUserUnlocked(userID string) error {
	log.Log(log.RUN_PREFIX, "Cleaning volume for user: %s...", userID)

	if err := a.unmountDiskTree(filepath.Join(a.Config.Volumes.Host.Homes, userID)); err != nil {
		return fmt.Errorf("unmount home disk for %s: %w", userID, err)
	}

	userHome := filepath.Join(a.Config.Volumes.Host.Homes, userID)
	userImg := filepath.Join(a.Config.Volumes.Host.Homes, userID+".img")

	homeDev, err := a.findLoopDevicesForImages(userImg)
	if err != nil {
		return fmt.Errorf("failed to find loop devices for user %s: %w", userID, err)
	}

	for _, dev := range homeDev {
		log.Log(log.DETAIL_PREFIX, "Detaching loop device: %s", dev)
		err = a.detachLoopDevice(dev)
		if err != nil {
			return fmt.Errorf("detach loop device %s: %w", dev, err)
		}
	}

	if err := a.systemAPI.RemoveAll(userHome); err != nil {
		return fmt.Errorf("failed to remove home dir for user %s: %w", userID, err)
	}

	if err := a.systemAPI.Remove(userImg); err != nil {
		return fmt.Errorf("failed to remove home image for user %s: %w", userID, err)
	}

	log.Log(log.DETAIL_PREFIX, "Volume clean completed for user: %s.", userID)

	return nil
}

// ensureDiskAll creates and mounts disks for all users and shared volumes.
func (a *App) ensureDiskAll() error {
	return a.withDiskLock(a.ensureDiskAllUnlocked)
}

func (a *App) ensureDiskAllUnlocked() error {
	if err := a.systemAPI.MkdirAll(a.Config.Volumes.Host.Homes, 0755); err != nil {
		return err
	}

	if err := a.createSharedDisk(a.Config.Volumes.Host.Share); err != nil {
		return err
	}
	if err := a.createSharedDisk(a.Config.Volumes.Host.Readonly); err != nil {
		return err
	}

	for userID := range a.UserIDs {
		if err := a.createUserDisk(userID, a.Config.ManagerService.AdminID == userID); err != nil {
			return err
		}
	}

	return nil
}

// ensureDiskUser creates and mounts disks for the specified user.
func (a *App) ensureDiskUser(userID string) error {
	if !ruleset.AllowedUserID(userID) {
		return fmt.Errorf("invalid user ID")
	}
	return a.withDiskLock(func() error { return a.ensureDiskUserUnlocked(userID) })
}

func (a *App) ensureDiskUserUnlocked(userID string) error {
	if !user.ExistsUser(a.UserIDs, userID) {
		return fmt.Errorf("user ID not found in auth list: %s", userID)
	}

	if err := a.systemAPI.MkdirAll(a.Config.Volumes.Host.Homes, 0755); err != nil {
		return err
	}

	if err := a.createSharedDisk(a.Config.Volumes.Host.Share); err != nil {
		return err
	}
	if err := a.createSharedDisk(a.Config.Volumes.Host.Readonly); err != nil {
		return err
	}

	if err := a.createUserDisk(userID, a.Config.ManagerService.AdminID == userID); err != nil {
		return err
	}

	return nil
}

// createSharedDisk creates and mounts a shared loopback disk at the target path.
func (a *App) createSharedDisk(path string) (retErr error) {
	sizeStr := a.Config.Volumes.DiskLimit
	size, err := convert.BytesFromString(sizeStr)
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
		log.Log(log.INFO_PREFIX, "Already mounted: %s", mountPoint)
		return nil
	}

	if exists, err := a.systemAPI.Exists(img); err != nil {
		return fmt.Errorf("failed to stat image file %s: %w", img, err)
	} else if !exists {
		log.Log(log.DETAIL_PREFIX, "Creating shared disk for %s (%s)", mountPoint, sizeStr)

		if err := a.createFormattedImage(img, size); err != nil {
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
		if retErr != nil {
			retErr = errors.Join(retErr, a.rollbackDiskMount(loopdev, mountPoint, mounted))
		}
	}()

	if err = a.systemAPI.Mount(loopdev, mountPoint); err != nil {
		return err
	}
	mounted = true

	return a.setDiskPermissions(mountPoint)
}

// createUserDisk creates and mounts a per-user loopback disk.
func (a *App) createUserDisk(userID string, isAdmin bool) (retErr error) {
	if !ruleset.AllowedUserID(userID) {
		return fmt.Errorf("invalid user ID: %q", userID)
	}
	sizeStr := a.Config.UserService.Limits.User.Disk
	if isAdmin {
		sizeStr = a.Config.UserService.Limits.Admin.Disk
	}

	size, err := convert.BytesFromString(sizeStr)
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
		log.Log(log.INFO_PREFIX, "Already mounted: %s", mountPoint)
		return nil
	}

	if exists, err := a.systemAPI.Exists(img); err != nil {
		return fmt.Errorf("failed to stat image file %s: %w", img, err)
	} else if !exists {
		log.Log(log.DETAIL_PREFIX, "Creating disk for %s (%s)", userID, sizeStr)

		if err := a.createFormattedImage(img, size); err != nil {
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
		if retErr != nil {
			retErr = errors.Join(retErr, a.rollbackDiskMount(loopdev, mountPoint, mounted))
		}
	}()

	if err = a.systemAPI.Mount(loopdev, mountPoint); err != nil {
		return err
	}
	mounted = true

	return a.setDiskPermissions(mountPoint)
}

// createFormattedImage removes only the new image when formatting fails, so a
// retry can initialize it again. Existing images never pass through this path.
func (a *App) createFormattedImage(img string, size int64) error {
	if err := a.systemAPI.CreateEmptyFile(img, size); err != nil {
		return err
	}
	if err := a.systemAPI.FormatExt4(img); err != nil {
		return errors.Join(err, a.systemAPI.Remove(img))
	}
	return nil
}

func (a *App) setDiskPermissions(path string) error {
	if err := a.systemAPI.Chown(path, a.Config.UserService.Runtime.UID, a.Config.UserService.Runtime.GID); err != nil {
		return err
	}
	return a.systemAPI.Chmod(path, 0755)
}

func (a *App) rollbackDiskMount(loopdev, mountPoint string, mounted bool) error {
	if mounted {
		if err := a.systemAPI.Unmount(mountPoint); err != nil {
			// The loop device still backs a mounted filesystem; leave it attached.
			return fmt.Errorf("rollback unmount %s: %w", mountPoint, err)
		}
	}
	if err := a.systemAPI.DetachLoopDevice(loopdev); err != nil {
		return fmt.Errorf("rollback detach %s: %w", loopdev, err)
	}
	return nil
}

// listMountedDirsDeepestFirst returns mounted directories under root from deepest to shallowest.
func (a *App) listMountedDirsDeepestFirst(root string) ([]string, error) {
	dirs, err := a.systemAPI.MountPointsUnder(root)
	if err != nil {
		return nil, err
	}

	sort.Slice(dirs, func(i, j int) bool {
		return len(dirs[i]) > len(dirs[j])
	})

	return dirs, nil
}

// findLoopDevicesForImages finds loop devices associated with image files under the specified directory.
func (a *App) findLoopDevicesForImages(homesDir string) ([]string, error) {
	return a.systemAPI.FindLoopDevicesForImages(homesDir)
}

// umountDisk unmounts the specified mount point if it is currently mounted.
func (a *App) umountDisk(mountPoint string) error {
	mounted, err := a.systemAPI.IsMountPoint(mountPoint)
	if err != nil {
		return err
	}

	if mounted {
		log.Log(log.DETAIL_PREFIX, "Unmounting: %s", mountPoint)
		return a.systemAPI.Unmount(mountPoint)
	}

	return nil
}

// detachLoopDevice detaches the specified loop device.
func (a *App) detachLoopDevice(dev string) error {
	return a.systemAPI.DetachLoopDevice(dev)
}

// Nested bind mounts must be released before their parent filesystem.
func (a *App) unmountDiskTree(root string) error {
	mounted, err := a.listMountedDirsDeepestFirst(root)
	if err != nil {
		return err
	}
	for _, path := range mounted {
		if err := a.umountDisk(path); err != nil {
			return err
		}
	}
	return a.umountDisk(root)
}
