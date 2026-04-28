package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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

// Option represents a runtime operation selected from CLI arguments.
type Option int

const (
	UP Option = iota
	DOWN
	RESTART
	VOLUME_CLEAN
	ENSURE_DISK
	PS
)

// Options stores parsed CLI flags and execution intents.
type Options struct {
	// Opts contains the ordered list of requested operations.
	Opts []Option
	// IsHelp indicates whether the help text should be printed.
	IsHelp bool
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

	opts, err := parseArgs(os.Args[1:])
	if err != nil {
		return err
	}

	if opts.IsHelp {
		printUsage(os.Args[0])
		return nil
	}

	app, err := app.CreateApp(currentDir, execPath, repoDir, sourceDir, configFile)
	if err != nil {
		return err
	}

	if err := app.LoadConfig(); err != nil {
		return err
	}

	if err := config.ValidateConfig(&app.Config); err != nil {
		return err
	}

	for _, v := range opts.Opts {
		switch v {
		case UP:
			if err := app.LoadUserList(); err != nil {
				return err
			}
			if err := app.PrepareUserDisks(); err != nil {
				return err
			}
			if err := app.ServiceUp(); err != nil {
				return err
			}

		case DOWN:
			if err := app.ServiceDown(); err != nil {
				return err
			}

		case RESTART:
			if err := app.LoadUserList(); err != nil {
				return err
			}
			if err := app.PrepareUserDisks(); err != nil {
				return err
			}
			if err := app.ServiceRestart(); err != nil {
				return err
			}

		case VOLUME_CLEAN:
			if err := app.VolumeClean(); err != nil {
				return err
			}

		case ENSURE_DISK:
			if err := app.LoadUserList(); err != nil {
				return err
			}

			if err := app.PrepareUserDisks(); err != nil {
				return err
			}

		case PS:
			if err := app.ServicePS(); err != nil {
				return err
			}
		}
	}

	return nil
}

// parseArgs converts CLI arguments into executable options.
func parseArgs(args []string) (Options, error) {
	opts := Options{
		Opts: make([]Option, 0),
	}
	if len(args) == 0 {
		return opts, errors.New(usageText(os.Args[0]))
	}

	for _, arg := range args {
		switch arg {
		case "-u", "up":
			opts.Opts = append(opts.Opts, UP)
		case "-d", "down":
			opts.Opts = append(opts.Opts, DOWN)
		case "-r", "restart":
			opts.Opts = append(opts.Opts, RESTART)
		case "-v", "volume-clean":
			opts.Opts = append(opts.Opts, VOLUME_CLEAN)
		case "-e", "ensure-disk":
			opts.Opts = append(opts.Opts, ENSURE_DISK)
		case "-p", "ps":
			opts.Opts = append(opts.Opts, PS)
		case "-h", "help":
			opts.IsHelp = true
		default:
			return opts, fmt.Errorf("invalid parameter: %s\n\n%s", arg, usageText(os.Args[0]))
		}
	}

	return opts, nil
}

// usageText returns the formatted help text for the CLI.
func usageText(bin string) string {
	result := ""
	result += "Usage:\n"
	result += fmt.Sprintf("  %s [OPTION]...\n\n", bin)
	result += "Options:\n"
	result += fmt.Sprintf("  %-25s# Show help message\n", "-h, help")
	result += fmt.Sprintf("  %-25s# Build images and start services\n", "-u, up")
	result += fmt.Sprintf("  %-25s# Stop and remove services\n", "-d, down")
	result += fmt.Sprintf("  %-25s# Restart services\n", "-r, restart")
	result += fmt.Sprintf("  %-25s# Reset all user directories\n", "-v, volume-clean")
	result += fmt.Sprintf("  %-25s# Create missing user directories and activate signed-up users\n", "-e, ensure-disk")
	result += fmt.Sprintf("  %-25s# Show service status\n\n", "-p, ps")
	result += "Examples:\n"
	result += fmt.Sprintf("  %-25s# Build and start\n", fmt.Sprintf("%s -u", bin))
	result += fmt.Sprintf("  %-25s# Restart and show status\n", fmt.Sprintf("%s -r -p", bin))
	result += fmt.Sprintf("  %-25s# Reset all user data\n\n", fmt.Sprintf("%s -v", bin))
	result += "Log Format:\n"
	result += fmt.Sprintf("  %s: Run messages indicating the start of major operations\n", format.RUN_PREFIX)
	result += fmt.Sprintf("  %s: Detail messages indicating the execution of specific steps\n", format.DETAIL_PREFIX)
	result += fmt.Sprintf("  %s: Informational messages about the progress and status of operations\n", format.INFO_PREFIX)
	result += fmt.Sprintf("  %s: Warning messages indicating potential issues that do not stop execution\n", format.WARNING_PREFIX)
	result += fmt.Sprintf("  %s: Error messages indicating failures that may require user attention\n", format.ERROR_PREFIX)
	return result
}

// printUsage prints the CLI help text.
func printUsage(bin string) {
	fmt.Println(usageText(bin))
}
