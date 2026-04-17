package app

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
	"github.com/elecbug/linuxus/src/ctl/internal/config"
)

// App stores runtime state and dependencies for service orchestration.
type App struct {
	// dockerClient is the Docker API client used for image/container/network operations.
	dockerClient *client.Client
	// context is the base context used for Docker API requests.
	context context.Context

	// currentDir is the working directory where the command was invoked.
	currentDir string
	// execPath is the absolute path of the current executable.
	execPath string
	// repoDir is the root directory where runtime assets are resolved.
	repoDir string
	// sourceDir is the root source directory used for relative config paths.
	sourceDir string
	// configFile is the absolute path to the runtime YAML configuration file.
	configFile string

	// Config stores the loaded runtime configuration.
	Config config.Config
	// UserIDs stores raw user IDs loaded from the auth list.
	UserIDs []string
	// SafeIDs stores sanitized IDs derived from UserIDs.
	SafeIDs []string
	// seen tracks already loaded user IDs to avoid duplicates.
	seen map[string]struct{}
}

// ContainerLimits contains resource limit values for a runtime container.
type ContainerLimits struct {
	// Memory is the memory limit string (for example, 512m).
	Memory string
	// CPUs is the CPU limit string used to compute NanoCPUs.
	CPUs string
	// Pids is the maximum number of processes allowed.
	Pids int
	// NofileSoft is the soft limit for open file descriptors.
	NofileSoft int
	// NofileHard is the hard limit for open file descriptors.
	NofileHard int
}

// RuntimeContainerSpec describes how a managed runtime container should be created.
type RuntimeContainerSpec struct {
	// Image is the Docker image reference used to create the container.
	Image string
	// Name is the container name.
	Name string
	// Hostname is the hostname assigned inside the container.
	Hostname string
	// WorkingDir is the working directory set for the container process.
	WorkingDir string
	// User is the Linux user (uid:gid or name) used to run the container process.
	User string
	// ReadOnly enables a read-only root filesystem when true.
	ReadOnly bool
	// Tmpfs defines tmpfs mount points and options.
	Tmpfs []string
	// Environment lists environment variables in KEY=VALUE form.
	Environment []string
	// Volumes lists bind mounts in Docker bind syntax.
	Volumes []string
	// Ports lists port bindings in HOST:CONTAINER format.
	Ports []string
	// Restart is the Docker restart policy name.
	Restart string
	// SecurityOpt lists Docker security options.
	SecurityOpt []string
	// CapDrop lists Linux capabilities dropped from the container.
	CapDrop []string
	// Limits defines runtime resource limits for the container.
	Limits ContainerLimits
	// Networks lists target networks to connect after create.
	Networks []string
}

// RuntimeNetworkSpec describes how a managed Docker network should be created.
type RuntimeNetworkSpec struct {
	// Name is the Docker network name.
	Name string
	// Subnet is the CIDR range assigned to the network.
	Subnet string
}

// CreateApp initializes Docker dependencies and returns an application runtime instance.
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
