package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/elecbug/linuxus/src/ctl/internal/app"
)

// Option represents a runtime operation selected from CLI arguments.
type Option int

const (
	UP Option = iota
	DOWN
	RESTART
	VOLUME_CLEAN
	PS
)

// Options stores parsed CLI flags and execution intents.
type Options struct {
	// Opts contains the ordered list of requested operations.
	Opts []Option
	// IsHelp indicates whether the help text should be printed.
	IsHelp bool
}

// main executes the CLI entrypoint and prints user-friendly errors.
func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
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
	configFile := filepath.Join(repoDir, "config.yml")

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
	if err := app.ValidateConfig(); err != nil {
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
	return fmt.Sprintf(`Usage:
  %s [OPTION]...

Options:
  -h, help          Show this help message
  -u, up            Build images and start runtime services
  -d, down          Stop and remove all runtime services
  -r, restart       Restart runtime services
  -v, volume-clean  Reset all user directories
  -p, ps            Show the status of all runtime services

Examples:
  %s -u
  %s -d
  %s -r`, bin, bin, bin, bin)
}

// printUsage prints the CLI help text.
func printUsage(bin string) {
	fmt.Println(usageText(bin))
}
