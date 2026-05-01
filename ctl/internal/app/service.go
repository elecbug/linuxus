package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/elecbug/linuxus/ctl/internal/format"
)

var ALL_USER_KEYWORDS = []string{"--all", "-a"}

// isAllUsersKeyword checks if the provided string matches any of the defined keywords for "all users".
func isAllUsersKeyword(s string) bool {
	for _, keyword := range ALL_USER_KEYWORDS {
		if s == keyword {
			return true
		}
	}
	return false
}

// ServiceUp builds images and starts all runtime-managed services.
func (a *App) ServiceUp() error {
	format.Log(format.RUN_PREFIX, "Starting runtime-managed containers...")

	if err := a.buildRuntimeImages(); err != nil {
		return err
	}
	if err := a.ensureRuntimeNetworks(); err != nil {
		return err
	}
	if err := a.ensureManagerContainer(); err != nil {
		return err
	}
	if err := a.ensureAuthContainer(); err != nil {
		return err
	}

	format.Log(format.DETAIL_PREFIX, "Runtime services started.")
	return nil
}

// ServiceDown stops and removes all runtime-managed services.
func (a *App) ServiceDown() error {
	format.Log(format.RUN_PREFIX, "Stopping runtime-managed containers...")
	if err := a.removeManagedContainers(); err != nil {
		return err
	}
	if err := a.removeManagedNetworks(); err != nil {
		return err
	}

	format.Log(format.DETAIL_PREFIX, "Runtime services stopped.")
	return nil
}

// ServiceRestart recreates runtime-managed services.
func (a *App) ServiceRestart() error {
	format.Log(format.RUN_PREFIX, "Restarting runtime-managed containers...")
	if err := a.ServiceDown(); err != nil {
		return err
	}
	return a.ServiceUp()
}

// VolumeClean unmounts and removes managed volume data and loop devices.
func (a *App) VolumeClean(userID string) error {
	if !isAllUsersKeyword(userID) {
		format.Log(format.RUN_PREFIX, "Cleaning volume for user: %s...", userID)

		if err := a.umountDisk(filepath.Join(a.Config.Volumes.Host.Homes, userID)); err != nil {
			format.Log(format.ERROR_PREFIX, "Failed to unmount home disk for user %s: %v", userID, err)
		}

		userHome := filepath.Join(a.Config.Volumes.Host.Homes, userID)
		userImg := filepath.Join(a.Config.Volumes.Host.Homes, userID+".img")

		homeDev, err := a.findLoopDevicesForImages(userHome)
		if err != nil {
			return fmt.Errorf("failed to find loop devices for user %s: %w", userID, err)
		}

		for _, dev := range homeDev {
			format.Log(format.DETAIL_PREFIX, "Detaching loop device: %s", dev)
			err = a.detachLoopDevice(dev)
			if err != nil {
				format.Log(format.ERROR_PREFIX, "Failed to detach loop device %s: %v", dev, err)
				continue
			}
		}

		if err := a.systemAPI.RemoveAll(userHome); err != nil {
			return fmt.Errorf("failed to remove home dir for user %s: %w", userID, err)
		}

		if err := a.systemAPI.Remove(userImg); err != nil {
			return fmt.Errorf("failed to remove home image for user %s: %w", userID, err)
		}

		format.Log(format.DETAIL_PREFIX, "Volume clean completed for user: %s.", userID)

		return nil
	} else {
		yes, err := format.Input("Are you sure you want to clean volumes for ALL users? This action cannot be undone. (yes/no): ")
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}

		if strings.ToLower(yes) != "yes" {
			format.Log(format.INFO_PREFIX, "Volume clean cancelled.")
			return nil
		}

		format.Log(format.RUN_PREFIX, "Cleaning volumes for all users...")

		_ = a.ServiceDown()

		homeMounts, err := a.listMountedDirsDeepestFirst(a.Config.Volumes.Host.Homes)
		if err != nil {
			return err
		}
		for _, dir := range homeMounts {
			err = a.umountDisk(dir)
			if err != nil {
				format.Log(format.ERROR_PREFIX, "Failed to unmount home disk at %s: %v", dir, err)
				continue
			}
		}

		for _, mountPoint := range []string{a.Config.Volumes.Host.Share, a.Config.Volumes.Host.Readonly} {
			err = a.umountDisk(mountPoint)
			if err != nil {
				format.Log(format.ERROR_PREFIX, "Failed to unmount shared disk at %s: %v", mountPoint, err)
				continue
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
			devs, err := a.findLoopDevicesForImages(filepath.Dir(mountPoint))
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
			format.Log(format.DETAIL_PREFIX, "Detaching loop device: %s", dev)
			err = a.detachLoopDevice(dev)
			if err != nil {
				format.Log(format.ERROR_PREFIX, "Failed to detach loop device %s: %v", dev, err)
				continue
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
		if err := a.systemAPI.RemoveAll(a.Config.Volumes.Host.Volumes); err != nil {
			return fmt.Errorf("failed to remove volumes dir: %w", err)
		}

		format.Log(format.DETAIL_PREFIX, "Volume clean completed.")
		return nil
	}
}

// ServicePS prints runtime status for managed containers and networks.
func (a *App) ServicePS(params []string) error {
	format.Log(format.RUN_PREFIX, "Gathering runtime status...")

	if len(params) == 0 {
		if err := a.showContainerInfos(); err != nil {
			return err
		}
		if err := a.showNetworkInfos(); err != nil {
			return err
		}
	} else if len(params) == 1 {
		switch params[0] {
		case "container", "c":
			if err := a.showContainerInfos(); err != nil {
				return err
			}
		case "network", "n":
			if err := a.showNetworkInfos(); err != nil {
				return err
			}
		case "all", "a":
			if err := a.showContainerInfos(); err != nil {
				return err
			}
			if err := a.showNetworkInfos(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid parameter for ps option: %s", params[0])
		}
	}

	return nil
}

// EnsureDisk creates and mounts shared/admin/user disk images.
func (a *App) EnsureDisk(userID string) error {
	if !isAllUsersKeyword(userID) {
		if !a.existsUser(userID) {
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
	} else {
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
			if err := a.createUserDisk(userID, a.Config.ManagerService.AdminID == userID); err != nil {
				return err
			}
		}

		return nil
	}
}

// AddUser adds a new user by creating necessary directories and activating the user in the system.
func (a *App) AddUser(userID string) error {
	format.Log(format.RUN_PREFIX, "Adding a new user...")

	if a.existsUser(userID) {
		return fmt.Errorf("user ID already exists: %s", userID)
	}

	password, err := format.InputPassword("Enter password for new user %s: ", userID)
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	cfmPassword, err := format.InputPassword("Confirm password for new user %s: ", userID)
	if err != nil {
		return fmt.Errorf("failed to read password confirmation: %w", err)
	}

	if password != cfmPassword {
		return fmt.Errorf("passwords do not match")
	}

	if err := a.updateUser(userID, password); err != nil {
		return fmt.Errorf("failed to add user: %w", err)
	}

	if err := a.createUserDisk(userID, a.Config.ManagerService.AdminID == userID); err != nil {
		return fmt.Errorf("failed to create user disk: %w", err)
	}

	format.Log(format.DETAIL_PREFIX, "User %s added successfully.", userID)

	return nil
}

// RemoveUser removes an existing user by deactivating the user in the system and cleaning up associated resources.
func (a *App) RemoveUser(userID string) error {
	format.Log(format.RUN_PREFIX, "Removing an existing user...")

	if !a.existsUser(userID) {
		return fmt.Errorf("user ID does not exist: %s", userID)
	}

	yes, err := format.Input("Are you sure you want to remove user %s? This action cannot be undone. (yes/no): ", userID)
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}

	if strings.ToLower(yes) != "yes" {
		format.Log(format.INFO_PREFIX, "User removal cancelled.")
		return nil
	}

	if err := a.removeUser(userID); err != nil {
		return fmt.Errorf("failed to remove user: %w", err)
	}

	if err := a.VolumeClean(userID); err != nil {
		return fmt.Errorf("failed to clean user volumes: %w", err)
	}

	format.Log(format.DETAIL_PREFIX, "User %s removed successfully.", userID)

	return nil
}
