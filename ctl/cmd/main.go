package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/elecbug/linuxus/src/ctl/internal/app"
)

type Option int

const (
	UP Option = iota
	DOWN
	RESTART
	VOLUME_CLEAN
	PS
)

type Options struct {
	Opts   []Option
	IsHelp bool
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

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
	configFile := filepath.Join(sourceDir, "config.yml")

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
			if err := app.LoadUsers(); err != nil {
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
			if err := app.LoadUsers(); err != nil {
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

func usageText(bin string) string {
	return fmt.Sprintf(`Usage:
  %s [OPTION]...

Options:
  -h, help          Show this help message
  -u, up            Build images and start runtime services
  -d, down          Stop and remove runtime services
  -r, restart       Restart runtime services
  -v, volume-clean  Reset all user directories
  -p, ps            Show the status of runtime services

Examples:
  %s -u
  %s -d
  %s -r`, bin, bin, bin, bin)
}

func printUsage(bin string) {
	fmt.Println(usageText(bin))
}
