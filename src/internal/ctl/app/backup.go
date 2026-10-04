package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	archive "github.com/elecbug/linuxus/src/internal/common/backup"
	"github.com/elecbug/linuxus/src/internal/common/config"
	"github.com/elecbug/linuxus/src/internal/common/system_api"
	"github.com/elecbug/linuxus/src/internal/common/user"
	"github.com/elecbug/linuxus/src/internal/ctl/log"
)

func VerifyBackup(path string, output io.Writer) error {
	file, err := os.Open(path)
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
func (a *App) maintainUser(id string, operation func() error) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("disk backup/restore requires root")
	}
	users, err := user.LoadUsers(a.Config.AuthService.Mounts.HostAuthListPath)
	if err != nil {
		return err
	}
	record, ok := users[id]
	if !ok {
		return fmt.Errorf("user does not exist")
	}
	wasLocked := user.IsLocked(record)
	if err := user.UpdateAccount(a.Config.AuthService.Mounts.HostAuthListPath, id, "lock", ""); err != nil {
		return err
	}
	if err := a.stopUserRuntime(id); err != nil {
		return fmt.Errorf("account remains locked; stop runtime: %w", err)
	}
	err = a.withDiskLock(func() error {
		if err := a.detachUserImage(id); err != nil {
			return err
		}
		return operation()
	})
	if err != nil {
		return fmt.Errorf("account %s remains locked; data retained for recovery: %w", id, err)
	}
	if !wasLocked {
		if err := user.UpdateAccount(a.Config.AuthService.Mounts.HostAuthListPath, id, "unlock", ""); err != nil {
			return err
		}
	}
	log.Log(log.DETAIL_PREFIX, "Maintenance complete for %s. Runtime is stopped; log in again to resume.", id)
	return nil
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
		image, err := os.Open(filepath.Join(a.Config.Volumes.Host.Homes, id+".img"))
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
	if err := a.outsideManagedStorage(input); err != nil {
		return err
	}
	imagePath := filepath.Join(a.Config.Volumes.Host.Homes, id+".img")
	if _, err := os.Lstat(imagePath); err == nil && !replace {
		return fmt.Errorf("existing image preserved; use --replace to restore over it")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	file, err := os.Open(input)
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
	manifest, err := archive.Read(file, temp, int64(available))
	if err != nil {
		return err
	}
	if manifest.UserID != id || manifest.UID != a.Config.UserService.Runtime.UID || manifest.GID != a.Config.UserService.Runtime.GID {
		return fmt.Errorf("backup user or UID/GID does not match this deployment")
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return a.maintainUser(id, func() error {
		previous := temp.Name() + ".previous"
		hasPrevious := false
		if _, err := os.Lstat(imagePath); err == nil {
			if !replace {
				return fmt.Errorf("image appeared during verification; use --replace")
			}
			if err := os.Rename(imagePath, previous); err != nil {
				return err
			}
			hasPrevious = true
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := os.Rename(temp.Name(), imagePath); err != nil {
			if hasPrevious {
				err = errors.Join(err, os.Rename(previous, imagePath))
			}
			return err
		}
		if err := a.createUserDisk(id, id == a.Config.ManagerService.AdminID); err != nil {
			if cleanupErr := a.detachUserImage(id); cleanupErr != nil {
				return errors.Join(err, fmt.Errorf("retain previous image at %s: %w", previous, cleanupErr))
			}
			if hasPrevious {
				if restoreErr := os.Rename(previous, imagePath); restoreErr != nil {
					return errors.Join(err, fmt.Errorf("previous image retained at %s: %w", previous, restoreErr))
				}
				return errors.Join(err, a.createUserDisk(id, id == a.Config.ManagerService.AdminID))
			}
			return err
		}
		if hasPrevious {
			return os.Remove(previous)
		}
		return nil
	})
}
