package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	archive "github.com/elecbug/linuxus/src/internal/common/backup"
	"github.com/elecbug/linuxus/src/internal/common/config"
	"github.com/elecbug/linuxus/src/internal/common/convert"
	"github.com/elecbug/linuxus/src/internal/common/ruleset"
	"github.com/elecbug/linuxus/src/internal/common/system_api"
	"github.com/elecbug/linuxus/src/internal/common/user"
	"github.com/elecbug/linuxus/src/internal/ctl/log"
)

func VerifyBackup(path string, output io.Writer) error {
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("backup must be a regular file")
	}
	manifest, err := archive.Read(file, nil, 0)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "Backup valid: user=%s bytes=%d uid=%d gid=%d created=%s SHA256=%s\n", manifest.UserID, manifest.Size, manifest.UID, manifest.GID, manifest.Created, manifest.SHA256)
	return err
}

func (a *App) outsideManagedStorage(path string) error {
	resolved, err := config.CanonicalStoragePath(path)
	if err != nil {
		return err
	}
	for _, root := range []string{a.Config.Volumes.Host.Homes, a.Config.Volumes.Host.Share, a.Config.Volumes.Host.Readonly} {
		base, err := config.CanonicalStoragePath(root)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(base, resolved)
		if err != nil {
			return err
		}
		if rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("backup must be outside managed user/shared disks")
		}
		if resolved == base+".img" {
			return fmt.Errorf("backup cannot replace a managed image")
		}
	}
	return nil
}

// Maintenance locks the account before crossing Manager's runtime lock, then
// takes the disk lock. Failure leaves the account locked for safe manual repair.
func (a *App) withMaintenanceLock(action func() error) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("disk maintenance requires root")
	}
	if err := os.MkdirAll(a.diskServiceDir(), 0700); err != nil {
		return err
	}
	lock, err := os.OpenFile(filepath.Join(a.diskServiceDir(), "maintenance.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("another backup/restore is active: %w", err)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	return action()
}

func (a *App) maintainUser(id string, operation func() error) error {
	return a.withMaintenanceLock(func() error { return a.maintainUserLocked(id, operation) })
}

// The caller holds the maintenance lock, including any restore verification.
func (a *App) maintainUserLocked(id string, operation func() error) (retErr error) {
	if err := a.checkPendingRestore(id); err != nil {
		return err
	}
	path := a.Config.AuthService.Mounts.HostAuthListPath
	if err := user.UpdateAccount(path, id, "maintenance", ""); err != nil {
		return err
	}
	defer func() {
		if retErr == nil {
			retErr = user.UpdateAccount(path, id, "maintenance-end", "")
		}
		if retErr != nil {
			retErr = fmt.Errorf("account %s remains under maintenance; inspect retained data and run recover-user before unlocking: %w", id, retErr)
		}
	}()
	if err := a.stopUserRuntime(id); err != nil {
		return err
	}
	if err := a.withDiskLock(func() error {
		if err := a.detachUserImage(id); err != nil {
			return err
		}
		return operation()
	}); err != nil {
		return err
	}
	log.Log(log.DETAIL_PREFIX, "Maintenance complete for %s. Runtime is stopped; log in again to resume.", id)
	return nil
}

// RecoverUser restores an interrupted replacement before clearing maintenance.
// Both host locks exclude active maintenance and disk preparation. Access stays locked.
func (a *App) RecoverUser(id string) error {
	if !ruleset.AllowedUserID(id) {
		return fmt.Errorf("invalid user ID")
	}
	return a.withMaintenanceLock(func() error {
		path := a.Config.AuthService.Mounts.HostAuthListPath
		users, err := user.LoadUsers(path)
		if err != nil {
			return err
		}
		record, exists := users[id]
		if !exists {
			return fmt.Errorf("user %q does not exist", id)
		}
		account, err := user.ParseAccount(record)
		if err != nil {
			return err
		}
		if !account.Maintenance {
			if !account.Locked {
				return fmt.Errorf("account is not under maintenance or locked")
			}
			if err := user.UpdateAccount(path, id, "maintenance", ""); err != nil {
				return err
			}
		}
		if err := a.stopUserRuntime(id); err != nil {
			return err
		}
		return a.withDiskLock(func() error {
			if err := a.recoverUserImage(id); err != nil {
				return err
			}
			return user.UpdateAccount(path, id, "maintenance-failed", "")
		})
	})
}

func (a *App) previousUserImage(id string) string {
	return filepath.Join(a.Config.Volumes.Host.Homes, ".restore-"+id+".previous")
}

func (a *App) checkPendingRestore(id string) error {
	if _, err := os.Lstat(a.previousUserImage(id)); err == nil {
		return fmt.Errorf("user %s has an interrupted restore; run recover-user first", id)
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func requireRegularImage(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return fmt.Errorf("image %s must be a nonempty regular file", path)
	}
	return nil
}

func syncImageDirectory(path string) error {
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

// Recover only existing images; never initialize missing data during recovery.
func (a *App) recoverUserImage(id string) error {
	imagePath := filepath.Join(a.Config.Volumes.Host.Homes, id+".img")
	previous := a.previousUserImage(id)
	hasPrevious := false
	if _, err := os.Lstat(previous); err == nil {
		if err := requireRegularImage(previous); err != nil {
			return err
		}
		hasPrevious = true
	} else if !os.IsNotExist(err) {
		return err
	}
	if !hasPrevious {
		if err := requireRegularImage(imagePath); err != nil {
			return fmt.Errorf("cannot recover missing or invalid home image: %w", err)
		}
	}
	if err := a.detachUserImage(id); err != nil {
		return err
	}
	if hasPrevious {
		if err := a.restorePreviousImage(id); err != nil {
			return err
		}
	}
	return a.createUserDisk(id, id == a.Config.ManagerService.AdminID)
}

func (a *App) restorePreviousImage(id string) error {
	previous := a.previousUserImage(id)
	image := filepath.Join(a.Config.Volumes.Host.Homes, id+".img")
	// Rename is a no-op for two hard links to the same inode. Remove the extra
	// name explicitly when interruption happened before the replacement.
	old, err := os.Stat(previous)
	if err != nil {
		return err
	}
	current, err := os.Lstat(image)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil && os.SameFile(old, current) {
		err = os.Remove(previous)
	} else {
		err = os.Rename(previous, image)
	}
	if err != nil {
		return fmt.Errorf("previous image retained at %s: %w", previous, err)
	}
	return syncImageDirectory(image)
}

func (a *App) detachUserImage(id string) error {
	mount := filepath.Join(a.Config.Volumes.Host.Homes, id)
	if err := a.unmountDiskTree(mount); err != nil {
		return err
	}
	devices, err := a.findLoopDevicesForImages(mount + ".img")
	if err != nil {
		return err
	}
	for _, device := range devices {
		if err := a.detachLoopDevice(device); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) BackupUser(id, output string) error {
	if !ruleset.AllowedUserID(id) {
		return fmt.Errorf("invalid user ID")
	}
	output, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	if err := a.outsideManagedStorage(output); err != nil {
		return err
	}
	if _, err := os.Lstat(output); err == nil {
		return fmt.Errorf("backup already exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(output), ".linuxus-backup-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	defer temp.Close()
	return a.maintainUser(id, func() error {
		image, err := os.OpenFile(filepath.Join(a.Config.Volumes.Host.Homes, id+".img"), os.O_RDONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			return err
		}
		defer image.Close()
		if err := archive.Write(temp, image, archive.Manifest{UserID: id, UID: a.Config.UserService.Runtime.UID, GID: a.Config.UserService.Runtime.GID}); err != nil {
			return err
		}
		if err := temp.Sync(); err != nil {
			return err
		}
		if err := temp.Close(); err != nil {
			return err
		}
		if err := os.Link(temp.Name(), output); err != nil {
			return err
		}
		// Mount again even when automatic activation is disabled.
		return a.createUserDisk(id, id == a.Config.ManagerService.AdminID)
	})
}

func (a *App) RestoreUser(id, input string, replace bool) error {
	return a.withMaintenanceLock(func() error { return a.restoreUserLocked(id, input, replace) })
}

func (a *App) restoreUserLocked(id, input string, replace bool) error {
	if !ruleset.AllowedUserID(id) {
		return fmt.Errorf("invalid user ID")
	}
	if err := a.outsideManagedStorage(input); err != nil {
		return err
	}
	imagePath := filepath.Join(a.Config.Volumes.Host.Homes, id+".img")
	if _, err := os.Lstat(imagePath); err == nil && !replace {
		return fmt.Errorf("existing image preserved; use --replace to restore over it")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	file, err := os.OpenFile(input, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("backup must be a regular file")
	}
	if err := os.MkdirAll(a.Config.Volumes.Host.Homes, 0755); err != nil {
		return err
	}
	if err := a.checkFreeSpace(); err != nil {
		return err
	}
	_, available, err := system_api.DiskSpace(a.Config.Volumes.Host.Homes)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(a.Config.Volumes.Host.Homes, ".restore-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	defer temp.Close()
	reserve := int64(0)
	if a.Config.Capacity.MinFreeSpace != "" {
		reserve, err = convert.BytesFromString(a.Config.Capacity.MinFreeSpace)
		if err != nil {
			return err
		}
	}
	if available <= uint64(reserve) {
		return fmt.Errorf("insufficient restore space above configured reserve")
	}
	manifest, err := archive.Read(file, temp, int64(available)-reserve)
	if err != nil {
		return err
	}
	if manifest.UserID != id || manifest.UID != a.Config.UserService.Runtime.UID || manifest.GID != a.Config.UserService.Runtime.GID {
		return fmt.Errorf("backup user or UID/GID does not match this deployment")
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return a.maintainUserLocked(id, func() error { return a.restorePreparedImage(id, temp.Name(), replace) })
}

// The caller holds maintenance and disk locks and has detached the old image.
func (a *App) restorePreparedImage(id, prepared string, replace bool) error {
	imagePath := filepath.Join(a.Config.Volumes.Host.Homes, id+".img")
	previous := a.previousUserImage(id)
	if err := a.checkPendingRestore(id); err != nil {
		return err
	}
	if err := requireRegularImage(prepared); err != nil {
		return err
	}
	hasPrevious := false
	if _, err := os.Lstat(imagePath); err == nil {
		if !replace {
			return fmt.Errorf("image appeared during verification; use --replace")
		}
		if err := requireRegularImage(imagePath); err != nil {
			return err
		}
		// Preserve the old inode before atomically replacing its public name.
		if err := os.Link(imagePath, previous); err != nil {
			return err
		}
		hasPrevious = true
		if err := syncImageDirectory(previous); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(prepared, imagePath); err != nil {
		if hasPrevious {
			err = errors.Join(err, a.restorePreviousImage(id))
		}
		return err
	}
	if err := syncImageDirectory(imagePath); err != nil {
		return err
	}
	if err := a.createUserDisk(id, id == a.Config.ManagerService.AdminID); err != nil {
		if cleanupErr := a.detachUserImage(id); cleanupErr != nil {
			return errors.Join(err, fmt.Errorf("retain previous image at %s: %w", previous, cleanupErr))
		}
		if hasPrevious {
			if restoreErr := a.restorePreviousImage(id); restoreErr != nil {
				return errors.Join(err, fmt.Errorf("previous image retained at %s: %w", previous, restoreErr))
			}
			return errors.Join(err, a.createUserDisk(id, id == a.Config.ManagerService.AdminID))
		}
		return err
	}
	if hasPrevious {
		if err := os.Remove(previous); err != nil {
			return err
		}
		return syncImageDirectory(imagePath)
	}
	return nil
}
