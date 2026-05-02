package app

import (
	"fmt"
	"strings"

	"github.com/elecbug/linuxus/ctl/internal/format"
)

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
func (a *App) VolumeClean(param string) error {
	if isKeyword(param) {
		if isAllUsersKeyword(param) {
			err := a.cleanVolumesAll()
			if err != nil {
				return fmt.Errorf("failed to clean volumes for all users: %w", err)
			}
			return nil
		} else {
			return fmt.Errorf("invalid parameter for volume clean option: %s, please use '--all' or '-a' to clean volumes for all users", param)
		}
	} else {
		err := a.cleanVolumeUser(param)
		if err != nil {
			return fmt.Errorf("failed to clean volume for user %s: %w", param, err)
		}
		return nil
	}
}

// EnsureDisk creates and mounts shared/admin/user disk images.
func (a *App) EnsureDisk(param string) error {
	if isKeyword(param) {
		if isAllUsersKeyword(param) {
			err := a.ensureDiskAll()
			if err != nil {
				return fmt.Errorf("failed to ensure disks for all users: %w", err)
			}
			return nil
		} else {
			return fmt.Errorf("invalid parameter for ensure disk option: %s, please use '--all' or '-a' to ensure disks for all users", param)
		}
	} else {
		err := a.ensureDiskUser(param)
		if err != nil {
			return fmt.Errorf("failed to ensure disk for user %s: %w", param, err)
		}
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
		switch strings.ToLower(params[0]) {
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
			return fmt.Errorf("invalid parameter for ps option: %s, please use 'container', 'network', or 'all'", params[0])
		}
	}

	return nil
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
