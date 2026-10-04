package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/elecbug/linuxus/src/internal/auth"
	"github.com/elecbug/linuxus/src/internal/common/config"
	"github.com/elecbug/linuxus/src/internal/common/ruleset"
	"github.com/elecbug/linuxus/src/internal/ctl/app"
	"github.com/elecbug/linuxus/src/internal/ctl/cli"
	"github.com/elecbug/linuxus/src/internal/ctl/log"
	"github.com/elecbug/linuxus/src/internal/manager"
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
	SERVE_DISKS
	INIT
	CONFIG_CHECK
	DOCTOR
	ACCOUNTS
	LIST_USERS
	TEMPLATES
	BACKUP
	RESTORE
	VERIFY_BACKUP
	SYSTEMD_UNIT
	SUPERVISE
)

// Options encapsulates the selected operations and their parameters.
type Options struct {
	Option  Opt
	Command string
	Params  *cli.Parameters
}

// run initializes the application and executes selected runtime operations.
func run() error {
	// Containers run the same binary without host-side configuration files.
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "serve-auth":
			return auth.Run()
		case "serve-manager":
			return manager.Run()
		}
	}

	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return err
	}

	opt, err := parseArgs(os.Args[0], os.Args[1:])
	if err != nil {
		return err
	}

	if opt.Option == HELP {
		fmt.Println(usageText(os.Args[0], true, true, true))
		return nil
	}

	configFile, err := config.ResolveConfigFile()
	if err != nil {
		return err
	}

	if opt.Option == INIT {
		if err := config.InitFile(configFile); err != nil {
			return err
		}
		log.Log(log.DETAIL_PREFIX, "Created %s from embedded defaults.", configFile)
		return nil
	}

	if opt.Option == CONFIG_CHECK {
		return app.CheckConfig(configFile, os.Stdout)
	}
	if opt.Option == DOCTOR {
		return app.Doctor(configFile, os.Stdout)
	}

	if opt.Option == SYSTEMD_UNIT {
		return app.WriteSystemdUnit(execPath, configFile, os.Stdout)
	}
	if opt.Option == VERIFY_BACKUP {
		return app.VerifyBackup(opt.Params.Params["file"], os.Stdout)
	}
	a, err := app.CreateApp(execPath, configFile)
	if err != nil {
		return err
	}

	defer a.Close()

	if err := a.LoadConfig(); err != nil {
		return err
	}

	if err := config.ValidateConfig(&a.Config); err != nil {
		return err
	}

	switch opt.Option {
	case LIST_USERS:
		return a.ListUsers(os.Stdout)
	case TEMPLATES:
		return a.ListTemplates(os.Stdout)
	case ACCOUNTS:
		return a.ServiceAccount(opt.Command, opt.Params)
	case BACKUP:
		return a.BackupUser(opt.Params.Params["user"], opt.Params.Params["output"])
	case RESTORE:
		return a.RestoreUser(opt.Params.Params["user"], opt.Params.Params["file"], opt.Params.Params["replace"] == cli.TRUE_STR)
	case SUPERVISE:
		return a.Supervise()
	case SERVE_DISKS:
		return a.ServeDisks()
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
	case "list-users":
		result.Option = LIST_USERS
	case "templates":
		result.Option = TEMPLATES
	case "lock-user", "unlock-user", "reset-password", "disconnect-user", "assign-template", "assign-class":
		result.Option = ACCOUNTS
	case "backup-user":
		result.Option = BACKUP
	case "restore-user":
		result.Option = RESTORE
	case "verify-backup":
		result.Option = VERIFY_BACKUP
	case "systemd-unit":
		result.Option = SYSTEMD_UNIT
	case "supervise":
		result.Option = SUPERVISE
	case "init":
		result.Option = INIT
	case "config-check":
		result.Option = CONFIG_CHECK
	case "doctor":
		result.Option = DOCTOR
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
	case "serve-disks":
		result.Option = SERVE_DISKS
	default:
		return result, fmt.Errorf("invalid parameter: '%s'\n\n%s", args[0], usageText(bin, true, true, false))
	}

	result.Command = args[0]
	params := make([]string, 0)

	if len(args) > 1 {
		params = append(params, args[1:]...)
	}

	parsed, err := cli.ParseParams(params)
	if err != nil {
		return result, err
	}
	result.Params = parsed
	switch result.Option {
	case UP, DOWN, RESTART, HELP, SERVE_DISKS, INIT, CONFIG_CHECK, DOCTOR, LIST_USERS, TEMPLATES, SYSTEMD_UNIT, SUPERVISE:
		if len(params) != 0 {
			return result, fmt.Errorf("%s does not accept arguments", args[0])
		}
	case ACCOUNTS, BACKUP, RESTORE, VERIFY_BACKUP:
		allowed := map[string]bool{}
		if result.Option != VERIFY_BACKUP {
			allowed["user"] = true
		}
		switch result.Option {
		case BACKUP:
			allowed["output"] = true
		case RESTORE:
			allowed["file"] = true
		case VERIFY_BACKUP:
			allowed["file"] = true
		case ACCOUNTS:
			if args[0] == "assign-template" {
				allowed["template"] = true
			}
			if args[0] == "assign-class" {
				allowed["class"] = true
			}
		}
		for key := range allowed {
			if parsed.Params[key] == "" {
				return result, fmt.Errorf("%s requires --%s", args[0], key)
			}
		}
		if result.Option == RESTORE {
			allowed["replace"] = true
		}
		for key := range parsed.Params {
			if !allowed[key] {
				return result, fmt.Errorf("unsupported --%s for %s", key, args[0])
			}
		}
		if parsed.MainParam != "" {
			return result, fmt.Errorf("unexpected positional argument")
		}
		if id := parsed.Params["user"]; id != "" && !ruleset.AllowedUserID(id) {
			return result, fmt.Errorf("invalid user ID")
		}
	case PS:
		if len(parsed.Params) != 0 {
			return result, fmt.Errorf("ps only accepts container, network, all, c, n, or a")
		}
		switch strings.ToLower(parsed.MainParam) {
		case "", "container", "network", "all", "c", "n", "a":
		default:
			return result, fmt.Errorf("ps only accepts container, network, all, c, n, or a")
		}
	case ADD_USER, REMOVE_USER, CLEAN_VOLUME, ENSURE_DISK:
		if parsed.MainParam != "" || len(parsed.Params) != 1 {
			return result, fmt.Errorf("%s requires exactly one --user <USERNAME> or supported --all option", args[0])
		}
		if id, ok := parsed.Params["user"]; ok {
			if !ruleset.AllowedUserID(id) {
				return result, fmt.Errorf("invalid user ID: %q", id)
			}
		} else if result.Option == ADD_USER || result.Option == REMOVE_USER {
			return result, fmt.Errorf("%s requires --user <USERNAME>", args[0])
		}
	}
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
		result += "Config: /etc/linuxus/.env (override with LINUXUS_CONFIG=/absolute/path/.env)\n\n"
		result += "Options:\n"
		result += "├─ General:\n"
		result += fmt.Sprintf("│  ├─ %-35s# Create /etc/linuxus/.env from embedded defaults\n", "init")
		result += fmt.Sprintf("│  ├─ %-35s# Validate settings without Docker or service changes\n", "config-check")
		result += fmt.Sprintf("│  ├─ %-35s# Diagnose host prerequisites without changing state\n", "doctor")
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
		result += fmt.Sprintf("├─ %-35s# Initialize deployment configuration\n", fmt.Sprintf("%s init", bin))
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
