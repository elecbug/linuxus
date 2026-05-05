package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/elecbug/linuxus/src/internal/ctl/app"
	"github.com/elecbug/linuxus/src/internal/ctl/cli"
	"github.com/elecbug/linuxus/src/internal/ctl/config"
	"github.com/elecbug/linuxus/src/internal/ctl/log"
)

// main executes the CLI entrypoint and prints user-friendly errors.
func main() {
	if err := run(); err != nil {
		log.Log(log.ERROR_PREFIX, "An error occurred: %v", err)
		os.Exit(1)
	}
}

// Opt represents a runtime operation selected from CLI arguments.
type Opt int

const (
	UP Opt = iota
	DOWN
	RESTART
	CLEAN_VOLUME
	ENSURE_DISK
	PS
	ADD_USER
	REMOVE_USER
	HELP
)

// Options encapsulates the selected operations and their parameters.
type Options struct {
	Option Opt
	Params *cli.Parameters
}

// run initializes the application and executes selected runtime operations.
func run() error {
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return err
	}

	repoDir := filepath.Dir(execPath)
	sourceDir := filepath.Join(repoDir, "src")
	configFile := filepath.Join(repoDir, "cfg", "config.yml")

	opt, err := parseArgs(os.Args[0], os.Args[1:])
	if err != nil {
		return err
	}

	if opt.Option == HELP {
		fmt.Println(usageText(os.Args[0], true, true, true))
		return nil
	}

	a, err := app.CreateApp(currentDir, execPath, repoDir, sourceDir, configFile)
	if err != nil {
		return err
	}

	if err := a.LoadConfig(); err != nil {
		return err
	}

	if err := config.ValidateConfig(&a.Config); err != nil {
		return err
	}

	switch opt.Option {
	case UP:
		if err := a.ServiceUp(opt.Params); err != nil {
			return err
		}

	case DOWN:
		if err := a.ServiceDown(opt.Params); err != nil {
			return err
		}

	case RESTART:
		if err := a.ServiceRestart(opt.Params); err != nil {
			return err
		}

	case CLEAN_VOLUME:
		if err := a.ServiceCleanVolume(opt.Params); err != nil {
			return err
		}

	case ENSURE_DISK:
		if err := a.ServiceEnsureDisk(opt.Params); err != nil {
			return err
		}

	case PS:
		if err := a.ServicePS(opt.Params); err != nil {
			return err
		}

	case ADD_USER:
		if err := a.ServiceAddUser(opt.Params); err != nil {
			return err
		}
	case REMOVE_USER:
		if err := a.ServiceRemoveUser(opt.Params); err != nil {
			return err
		}
	}

	return nil
}

// parseArgs converts CLI arguments into executable options.
func parseArgs(bin string, args []string) (Options, error) {
	result := Options{
		Params: cli.NewParameters(),
	}

	if len(args) == 0 {
		return result, errors.New(usageText(bin, true, true, true))
	}

	switch args[0] {
	case "up":
		result.Option = UP
	case "down":
		result.Option = DOWN
	case "restart":
		result.Option = RESTART
	case "clean-volume":
		result.Option = CLEAN_VOLUME
	case "ensure-disk":
		result.Option = ENSURE_DISK
	case "ps":
		result.Option = PS
	case "add-user":
		result.Option = ADD_USER
	case "remove-user":
		result.Option = REMOVE_USER
	case "help":
		result.Option = HELP
	default:
		return result, fmt.Errorf("invalid parameter: '%s'\n\n%s", args[0], usageText(bin, true, true, false))
	}

	params := make([]string, 0)

	if len(args) > 1 {
		params = append(params, args[1:]...)
	}

	result.Params = cli.ParseParams(params)

	return result, nil
}

// s returns "s" if n is not 1, otherwise returns an empty string. Used for pluralization in error messages.
func s(n int) string {
	if n == 0 || n == 1 {
		return ""
	}
	return "s"
}

// usageText returns the formatted help text for the CLI.
func usageText(bin string, showUsage, showExample, showLogFormat bool) string {
	result := ""
	if showUsage {
		result += "Usage: " + fmt.Sprintf("%s [OPTION]...\n", bin)
		result += "\n"
		result += "Options:\n"
		result += "├─ General:\n"
		result += fmt.Sprintf("│  └─ %-35s# Show help message\n", "help")
		result += "│\n"
		result += "├─ Service Management:\n"
		result += fmt.Sprintf("│  ├─ %-35s# Build images and start services\n", "up")
		result += fmt.Sprintf("│  ├─ %-35s# Stop and remove services\n", "down")
		result += fmt.Sprintf("│  ├─ %-35s# Restart services\n", "restart")
		result += fmt.Sprintf("│  └─ %-35s# Show status about linuxus service\n", "ps [OPTION]")
		result += fmt.Sprintf("│     %-35s  - OPTION can be one of container, network, all or their shorthand c, n, a. If not specified, defaults to all.\n", "")
		result += "│\n"
		result += "├─ User Management:\n"
		result += fmt.Sprintf("│  ├─ %-35s# Add a new user\n", "add-user --user <USERNAME>")
		result += fmt.Sprintf("│  └─ %-35s# Remove an existing user\n", "remove-user --user <USERNAME>")
		result += "│\n"
		result += "└─ Disk Management:\n"
		result += fmt.Sprintf("   ├─ %-35s# Remove all user directories if the option is all, otherwise remove specific user directory\n", "clean-volume <OPTION>")
		result += fmt.Sprintf("   │  %-35s  - OPTION can be --all, --user or their shorthand -a, -u. If --user is specified, a username must be provided.\n", "")
		result += fmt.Sprintf("   └─ %-35s# Create a missing user directory if the option is all, otherwise create a specific user directory\n", "ensure-disk <OPTION>")
		result += fmt.Sprintf("      %-35s  - OPTION can be --all, --user or their shorthand -a, -u. If --user is specified, a username must be provided.\n", "")
	}
	if showExample {
		result += "\n"
		result += "Examples:\n"
		result += fmt.Sprintf("├─ %-35s# Build and start\n", fmt.Sprintf("%s up", bin))
		result += fmt.Sprintf("├─ %-35s# Restart\n", fmt.Sprintf("%s restart", bin))
		result += fmt.Sprintf("└─ %-35s# Show network status of linuxus service\n", fmt.Sprintf("%s ps network", bin))
	}
	if showLogFormat {
		result += "\n"
		result += "Log Format:\n"
		result += fmt.Sprintf("  %s: Run messages indicating the start of major operations\n", log.RUN_PREFIX)
		result += fmt.Sprintf("  %s: Detail messages indicating the execution of specific steps\n", log.DETAIL_PREFIX)
		result += fmt.Sprintf("  %s: Informational messages about the progress and status of operations\n", log.INFO_PREFIX)
		result += fmt.Sprintf("  %s: Warning messages indicating potential issues that do not stop execution\n", log.WARNING_PREFIX)
		result += fmt.Sprintf("  %s: Error messages indicating failures that may require user attention\n", log.ERROR_PREFIX)
		result += fmt.Sprintf("  %s: Input messages indicating user input or interaction\n", log.INPUT_PREFIX)
	}
	return result
}
