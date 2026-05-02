package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/elecbug/linuxus/ctl/internal/app"
	"github.com/elecbug/linuxus/ctl/internal/config"
	"github.com/elecbug/linuxus/ctl/internal/format"
)

// main executes the CLI entrypoint and prints user-friendly errors.
func main() {
	if err := run(); err != nil {
		format.Log(format.ERROR_PREFIX, "An error occurred: %v", err)
		os.Exit(1)
	}
}

// Opt represents a runtime operation selected from CLI arguments.
type Opt int

const (
	UP Opt = iota
	DOWN
	RESTART
	VOLUME_CLEAN
	ENSURE_DISK
	PS
	ADD_USER
	REMOVE_USER
	HELP
)

// Options encapsulates the selected operations and their parameters.
type Options struct {
	Option Opt
	Params []string
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
		if err := a.LoadUserList(); err != nil {
			return err
		}
		if err := a.ServiceUp(); err != nil {
			return err
		}

	case DOWN:
		if err := a.ServiceDown(); err != nil {
			return err
		}

	case RESTART:
		if err := a.LoadUserList(); err != nil {
			return err
		}
		if err := a.ServiceRestart(); err != nil {
			return err
		}

	case VOLUME_CLEAN:
		if err := a.LoadUserList(); err != nil {
			return err
		}

		if err := a.VolumeClean(opt.Params[0]); err != nil {
			return err
		}

	case ENSURE_DISK:
		if err := a.LoadUserList(); err != nil {
			return err
		}

		if err := a.EnsureDisk(opt.Params[0]); err != nil {
			return err
		}

	case PS:
		if err := a.ServicePS(opt.Params); err != nil {
			return err
		}

	case ADD_USER:
		if err := a.LoadUserList(); err != nil {
			return err
		}

		if err := a.AddUser(opt.Params[0]); err != nil {
			return err
		}
	case REMOVE_USER:
		if err := a.LoadUserList(); err != nil {
			return err
		}

		if err := a.RemoveUser(opt.Params[0]); err != nil {
			return err
		}
	}

	return nil
}

// parseArgs converts CLI arguments into executable options.
func parseArgs(bin string, args []string) (Options, error) {
	result := Options{}

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
	case "volume-clean":
		result.Option = VOLUME_CLEAN
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

	if len(args) > 1 {
		result.Params = args[1:]
	}

	if ((result.Option == HELP || result.Option == UP || result.Option == DOWN || result.Option == RESTART) && len(result.Params) > 0) ||
		((result.Option == ADD_USER || result.Option == REMOVE_USER || result.Option == ENSURE_DISK || result.Option == VOLUME_CLEAN) && len(result.Params) != 1) ||
		(result.Option == PS && len(result.Params) > 1) {
		return result, fmt.Errorf("invalid parameter%s [%s] for '%s'\n\n%s",
			s(len(result.Params)),
			strings.Join(result.Params, " "),
			args[0],
			usageText(bin, true, true, false),
		)
	}

	return result, nil
}

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
		result += fmt.Sprintf("│  ├─ %-35s# Add a new user\n", "add-user <USERNAME>")
		result += fmt.Sprintf("│  └─ %-35s# Remove an existing user\n", "remove-user <USERNAME>")
		result += "│\n"
		result += "└─ Disk Management:\n"
		result += fmt.Sprintf("   ├─ %-35s# Remove all user directories if the option is all, otherwise remove specific user directory\n", "volume-clean <OPTION|USERNAME>")
		result += fmt.Sprintf("   │  %-35s  - OPTION can be --all or their shorthand -a. Otherwise, specify a username.\n", "")
		result += fmt.Sprintf("   └─ %-35s# Create a missing user directory if the option is all, otherwise create a specific user directory\n", "ensure-disk <OPTION|USERNAME>")
		result += fmt.Sprintf("      %-35s  - OPTION can be --all or their shorthand -a. Otherwise, specify a username.\n", "")
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
		result += fmt.Sprintf("  %s: Run messages indicating the start of major operations\n", format.RUN_PREFIX)
		result += fmt.Sprintf("  %s: Detail messages indicating the execution of specific steps\n", format.DETAIL_PREFIX)
		result += fmt.Sprintf("  %s: Informational messages about the progress and status of operations\n", format.INFO_PREFIX)
		result += fmt.Sprintf("  %s: Warning messages indicating potential issues that do not stop execution\n", format.WARNING_PREFIX)
		result += fmt.Sprintf("  %s: Error messages indicating failures that may require user attention\n", format.ERROR_PREFIX)
		result += fmt.Sprintf("  %s: Input messages indicating user input or interaction\n", format.INPUT_PREFIX)
	}
	return result
}
