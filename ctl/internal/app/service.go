package app

import (
	"fmt"
	"strings"

	"github.com/elecbug/linuxus/ctl/internal/cli"
	"github.com/elecbug/linuxus/ctl/internal/format"
)

// ServiceUp builds images and starts all runtime-managed services.
func (a *App) ServiceUp(params *cli.Parameters) error {
	if params != nil && (len(params.Params) > 0 || params.MainParam != "") {
		return fmt.Errorf("service up option does not accept any parameters, please remove any provided parameters")
	}

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
func (a *App) ServiceDown(params *cli.Parameters) error {
	if params != nil && (len(params.Params) > 0 || params.MainParam != "") {
		return fmt.Errorf("service down option does not accept any parameters, please remove any provided parameters")
	}

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
func (a *App) ServiceRestart(params *cli.Parameters) error {
	if params != nil && (len(params.Params) > 0 || params.MainParam != "") {
		return fmt.Errorf("service up option does not accept any parameters, please remove any provided parameters")
	}

	format.Log(format.RUN_PREFIX, "Restarting runtime-managed containers...")

	if err := a.ServiceDown(params); err != nil {
		return err
	}

	format.Log(format.INFO_PREFIX, "All services stopped. Restarting services...")
	if err := a.ServiceUp(params); err != nil {
		return err
	}

	format.Log(format.DETAIL_PREFIX, "Runtime services restarted.")
	return nil
}

// ServicePS prints runtime status for managed containers and networks.
func (a *App) ServicePS(params *cli.Parameters) error {
	if len(params.Params) > 0 {
		return fmt.Errorf("too many parameters for ps option, please specify only one of 'container', 'network', or 'all'")
	}

	format.Log(format.RUN_PREFIX, "Gathering runtime status...")
	param := params.MainParam

	if param == "" {
		if err := a.showContainerInfos(); err != nil {
			return err
		}
		if err := a.showNetworkInfos(); err != nil {
			return err
		}
	} else {
		switch strings.ToLower(param) {
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
			return fmt.Errorf("invalid parameter for ps option: %s, please use 'container', 'network', or 'all'", param)
		}
	}

	return nil
}

// ServiceCleanVolume unmounts and removes managed volume data and loop devices.
func (a *App) ServiceCleanVolume(params *cli.Parameters) error {
	if len(params.Params) > 1 || (len(params.Params) == 1 && params.MainParam != "") {
		return fmt.Errorf("too many parameters for volume clean option, please specify only one '--user <USERNAME>' or use '--all' to clean volumes for all users")
	}

	isAll, okAll := params.Params["all"]
	isA, okA := params.Params["a"]

	userID, okUser := params.Params["user"]
	uID, okU := params.Params["u"]

	if (okAll && isAll == cli.TRUE_STR) || (okA && isA == cli.TRUE_STR) {
		err := a.cleanVolumesAll()
		if err != nil {
			return fmt.Errorf("failed to clean volumes for all users: %w", err)
		}
		return nil
	} else if okUser || okU {
		if okU {
			userID = uID
		}

		err := a.cleanVolumeUser(userID)
		if err != nil {
			return fmt.Errorf("failed to clean volume for user %s: %w", userID, err)
		}
		return nil
	} else {
		return fmt.Errorf("invalid user ID parameter for volume clean option, expected a string value")
	}
}

// ServiceEnsureDisk creates and mounts shared/admin/user disk images.
func (a *App) ServiceEnsureDisk(params *cli.Parameters) error {
	if len(params.Params) > 1 || (len(params.Params) == 1 && params.MainParam != "") {
		return fmt.Errorf("too many parameters for ensure disk option, please specify only one '--user <USERNAME>' or use '--all' to ensure disks for all users")
	}

	isAll, okAll := params.Params["all"]
	isA, okA := params.Params["a"]

	userID, okUser := params.Params["user"]
	uID, okU := params.Params["u"]

	if (okAll && isAll == cli.TRUE_STR) || (okA && isA == cli.TRUE_STR) {
		err := a.ensureDiskAll()
		if err != nil {
			return fmt.Errorf("failed to ensure disks for all users: %w", err)
		}
		return nil
	} else if okUser || okU {
		if okU {
			userID = uID
		}

		err := a.ensureDiskUser(userID)
		if err != nil {
			return fmt.Errorf("failed to ensure disk for user %s: %w", userID, err)
		}
		return nil
	} else {
		return fmt.Errorf("invalid user ID parameter for ensure disk option, expected a string value")
	}
}

// ServiceAddUser adds a new user by creating necessary directories and activating the user in the system.
func (a *App) ServiceAddUser(params *cli.Parameters) error {
	if len(params.Params) != 1 || params.MainParam != "" {
		return fmt.Errorf("invalid number of parameters for add-user option, please specify exactly one '--user <USERNAME>' and no main parameter")
	}

	userID, okUser := params.Params["user"]
	uID, okU := params.Params["u"]

	if !okUser && !okU {
		return fmt.Errorf("missing --user <USERNAME> for add-user option")
	}

	if okU {
		userID = uID
	}

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

// ServiceRemoveUser removes an existing user by deactivating the user in the system and cleaning up associated resources.
func (a *App) ServiceRemoveUser(params *cli.Parameters) error {
	if len(params.Params) != 1 || params.MainParam != "" {
		return fmt.Errorf("invalid number of parameters for remove-user option, please specify exactly one '--user <USERNAME>' and no main parameter")
	}

	format.Log(format.RUN_PREFIX, "Removing an existing user...")

	userID, okUser := params.Params["user"]
	uID, okU := params.Params["u"]

	if !okUser && !okU {
		return fmt.Errorf("missing --user <USERNAME> for remove-user option")
	}

	if okU {
		userID = uID
	}

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

	if err := a.cleanVolumeUser(userID); err != nil {
		return fmt.Errorf("failed to clean user volumes: %w", err)
	}

	format.Log(format.DETAIL_PREFIX, "User %s removed successfully.", userID)

	return nil
}
