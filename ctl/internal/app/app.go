package app

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
	"github.com/elecbug/linuxus/src/ctl/internal/config"
)

type App struct {
	dockerClient *client.Client
	context      context.Context

	currentDir string
	execPath   string
	repoDir    string
	sourceDir  string
	configFile string

	Config  config.Config
	UserIDs []string
	SafeIDs []string
	seen    map[string]struct{}
}

func CreateApp(currentDir, execDir, repoDir, sourceDir, configFile string) (*App, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	app := &App{
		dockerClient: cli,
		context:      context.Background(),
		currentDir:   currentDir,
		execPath:     execDir,
		repoDir:      repoDir,
		sourceDir:    sourceDir,
		configFile:   configFile,
		seen:         make(map[string]struct{}),
	}

	return app, nil
}
