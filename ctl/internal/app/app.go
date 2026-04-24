package app

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
	"github.com/elecbug/linuxus/src/ctl/internal/config"
	"github.com/elecbug/linuxus/src/ctl/internal/system"
)

// App stores runtime state, config, and Docker dependencies for linuxusctl.
type App struct {
	// dockerClient is the shared Docker API client used by the CLI.
	dockerClient *client.Client
	// context is passed to Docker API calls.
	context context.Context
	// systemAPI abstracts OS-specific operations for better testability and error handling.
	systemAPI system.API

	// currentDir is the directory where the CLI command was executed.
	currentDir string
	// execPath is the absolute executable path for the running binary.
	execPath string
	// repoDir is the repository root resolved from the executable location.
	repoDir string
	// sourceDir points to the repository source directory.
	sourceDir string
	// configFile points to the runtime configuration file.
	configFile string

	// Config stores the parsed application configuration.
	Config config.Config
	// UserIDs stores raw user IDs parsed from auth list.
	UserIDs []string
	// SafeIDs stores sanitized user IDs safe for Docker resource names.
	SafeIDs []string
	// seen tracks deduplication of user IDs while parsing auth data.
	seen map[string]struct{}
}

// CreateApp creates an App instance and initializes the Docker client.
func CreateApp(currentDir, execDir, repoDir, sourceDir, configFile string) (*App, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	app := &App{
		dockerClient: cli,
		context:      context.Background(),
		systemAPI:    system.NewSystemAPI(),
		currentDir:   currentDir,
		execPath:     execDir,
		repoDir:      repoDir,
		sourceDir:    sourceDir,
		configFile:   configFile,
		seen:         make(map[string]struct{}),
	}

	return app, nil
}
