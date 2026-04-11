package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (a *App) PrepareUserDisks() error {
	if err := os.MkdirAll(a.Config.Volumes.Host.Homes, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(a.Config.Volumes.Host.Share, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(a.Config.Volumes.Host.Readonly, 0755); err != nil {
		return err
	}

	for _, userID := range a.UserIDs {
		if err := a.createUserDisk(userID, false); err != nil {
			return err
		}
	}
	if err := a.createUserDisk(a.Config.Admin.UserID, true); err != nil {
		return err
	}
	return nil
}

func (a *App) createUserDisk(userID string, isAdmin bool) error {
	size := a.Config.UserLimits.Disk
	if isAdmin {
		size = a.Config.AdminLimits.Disk
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
		fmt.Sprintf("%d:%d", a.Config.ContainerRuntime.UID, a.Config.ContainerRuntime.GID),
		mountPoint); err != nil {
		return err
	}
	if err = runCmd("sudo", "chmod", "755", mountPoint); err != nil {
		return err
	}

	return nil
}
