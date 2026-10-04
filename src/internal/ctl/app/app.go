package app

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
	"github.com/elecbug/linuxus/src/internal/common/config"
	"github.com/elecbug/linuxus/src/internal/common/system_api"
)

// App stores runtime state, config, and Docker dependencies for linuxusctl.
type App struct {
	// dockerClient is the shared Docker API client used by the CLI.
	dockerClient *client.Client
	// context is passed to Docker API calls.
	context context.Context
	// systemAPI abstracts OS-specific operations for better testability and error handling.
	systemAPI system_api.API

	// execPath is the absolute executable path for the running binary.
	execPath string
	// configFile points to the runtime configuration file.
	configFile string

	// Config stores the parsed application configuration.
	Config config.Config
	// UserIDs stores raw user IDs parsed from auth list.
	UserIDs map[string]string
}

// CreateApp creates an App instance and initializes the Docker client.
func CreateApp(execPath, configFile string) (*App, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	app := &App{
		dockerClient: cli,
		context:      context.Background(),
		systemAPI:    system_api.NewSystemAPI(),
		execPath:     execPath,
		configFile:   configFile,
		UserIDs:      nil,
	}

	return app, nil
}

// Close releases the Docker client resources.
func (a *App) Close() error {
	return a.dockerClient.Close()
}
