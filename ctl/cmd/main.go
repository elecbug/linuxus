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

	opt, err := parseArgs(os.Args[1:])
	if err != nil {
		return err
	}

	if opt.Option == HELP {
		printUsage(os.Args[0])
		return nil
	}

	appli, err := app.CreateApp(currentDir, execPath, repoDir, sourceDir, configFile)
	if err != nil {
		return err
	}

	if err := appli.LoadConfig(); err != nil {
		return err
	}

	if err := config.ValidateConfig(&appli.Config); err != nil {
		return err
	}

	switch opt.Option {
	case UP:
		if err := appli.LoadUserList(); err != nil {
			return err
		}
		if err := appli.PrepareUserDisks(app.ALL_USER_KEYWORDS[0]); err != nil {
			return err
		}
		if err := appli.ServiceUp(); err != nil {
			return err
		}

	case DOWN:
		if err := appli.ServiceDown(); err != nil {
			return err
		}

	case RESTART:
		if err := appli.LoadUserList(); err != nil {
			return err
		}
		if err := appli.PrepareUserDisks(app.ALL_USER_KEYWORDS[0]); err != nil {
			return err
		}
		if err := appli.ServiceRestart(); err != nil {
			return err
		}

	case VOLUME_CLEAN:
		if err := appli.LoadUserList(); err != nil {
			return err
		}

		if err := appli.VolumeClean(opt.Params[0]); err != nil {
			return err
		}

	case ENSURE_DISK:
		if err := appli.LoadUserList(); err != nil {
			return err
		}

		if err := appli.PrepareUserDisks(opt.Params[0]); err != nil {
			return err
		}

	case PS:
		if err := appli.ServicePS(opt.Params); err != nil {
			return err
		}

	case ADD_USER:
		if err := appli.LoadUserList(); err != nil {
			return err
		}

		if err := appli.AddUser(opt.Params[0]); err != nil {
			return err
		}
	case REMOVE_USER:
		if err := appli.LoadUserList(); err != nil {
			return err
		}

		if err := appli.RemoveUser(opt.Params[0]); err != nil {
			return err
		}
	}

	return nil
}

// parseArgs converts CLI arguments into executable options.
func parseArgs(args []string) (Options, error) {
	if len(args) == 0 {
		return Options{}, errors.New(usageText(os.Args[0]))
	}

	result := Options{}

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
		return Options{}, fmt.Errorf("invalid parameter: %s\n\n%s", args[0], usageText(os.Args[0]))
	}

	if len(args) > 1 {
		result.Params = args[1:]
	}

	if result.Option == HELP && len(result.Params) > 0 {
		return Options{}, fmt.Errorf("help option does not accept parameters: %s", strings.Join(result.Params, ""))
	} else if (result.Option == UP || result.Option == DOWN || result.Option == RESTART) && len(result.Params) > 0 {
		return Options{}, fmt.Errorf("option %s does not accept parameters: %s", args[0], strings.Join(result.Params, " "))
	} else if (result.Option == ADD_USER || result.Option == REMOVE_USER || result.Option == ENSURE_DISK || result.Option == VOLUME_CLEAN) && len(result.Params) != 1 {
		return Options{}, fmt.Errorf("option %s requires exactly one parameter: username", args[0])
	} else if result.Option == PS && len(result.Params) > 1 {
		return Options{}, fmt.Errorf("option %s accepts at most one parameter: %s", args[0], strings.Join(result.Params, " "))
	}

	return result, nil
}

// usageText returns the formatted help text for the CLI.
func usageText(bin string) string {
	result := ""
	result += "Usage: " + fmt.Sprintf("%s [OPTION]...\n", bin)
	result += "\n"
	result += "Options:\n"
	result += "├─ General:\n"
	result += fmt.Sprintf("│  └─ %-35s# Show help message\n", "help")
	result += "├─ Service Management:\n"
	result += fmt.Sprintf("│  ├─ %-35s# Build images and start services\n", "up")
	result += fmt.Sprintf("│  ├─ %-35s# Stop and remove services\n", "down")
	result += fmt.Sprintf("│  ├─ %-35s# Restart services\n", "restart")
	result += fmt.Sprintf("│  └─ %-35s# Show status about linuxus service\n", "ps [all|container|network]")
	result += "├─ User Management:\n"
	result += fmt.Sprintf("│  ├─ %-35s# Add a new user\n", "add-user <USERNAME>")
	result += fmt.Sprintf("│  └─ %-35s# Remove an existing user\n", "remove-user <USERNAME>")
	result += "└─ Disk Management:\n"
	result += fmt.Sprintf("   ├─ %-35s# Remove all user directories if the option is all, otherwise remove specific user directory\n", "volume-clean <OPTION|USERNAME>")
	result += fmt.Sprintf("   └─ %-35s# Create a missing user directory if the option is all, otherwise create a specific user directory\n", "ensure-disk <OPTION|USERNAME>")
	result += "\n"
	result += "Examples:\n"
	result += fmt.Sprintf("├─ %-35s# Build and start\n", fmt.Sprintf("%s up", bin))
	result += fmt.Sprintf("├─ %-35s# Restart\n", fmt.Sprintf("%s restart", bin))
	result += fmt.Sprintf("└─ %-35s# Show network status of linuxus service\n", fmt.Sprintf("%s ps network", bin))
	result += "\n"
	result += "Log Format:\n"
	result += fmt.Sprintf("  %s: Run messages indicating the start of major operations\n", format.RUN_PREFIX)
	result += fmt.Sprintf("  %s: Detail messages indicating the execution of specific steps\n", format.DETAIL_PREFIX)
	result += fmt.Sprintf("  %s: Informational messages about the progress and status of operations\n", format.INFO_PREFIX)
	result += fmt.Sprintf("  %s: Warning messages indicating potential issues that do not stop execution\n", format.WARNING_PREFIX)
	result += fmt.Sprintf("  %s: Error messages indicating failures that may require user attention\n", format.ERROR_PREFIX)
	result += fmt.Sprintf("  %s: Input messages indicating user input or interaction\n", format.INPUT_PREFIX)
	return result
}

// printUsage prints the CLI help text.
func printUsage(bin string) {
	fmt.Println(usageText(bin))
}
