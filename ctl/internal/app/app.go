package app

import (
	"context"
	"fmt"
	"os"

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

type ContainerLimits struct {
	Memory     string
	CPUs       string
	Pids       int
	NofileSoft int
	NofileHard int
}

type RuntimeContainerSpec struct {
	Image       string
	Name        string
	Hostname    string
	WorkingDir  string
	User        string
	ReadOnly    bool
	Tmpfs       []string
	Environment []string
	Volumes     []string
	Ports       []string
	Restart     string
	SecurityOpt []string
	CapDrop     []string
	Limits      ContainerLimits
	Networks    []string
}

type RuntimeNetworkSpec struct {
	Name   string
	Subnet string
}

func CreateApp(currentDir, execDir, repoDir, sourceDir, configFile string) (*App, error) {

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

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

	if err := os.Chdir(app.sourceDir); err != nil {
		return nil, fmt.Errorf("failed to change directory to source dir: %w", err)
	}
	defer func() {
		_ = os.Chdir(app.currentDir)
	}()

	return app, nil
}
