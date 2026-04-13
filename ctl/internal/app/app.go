package app

import (
	"path/filepath"

	"github.com/docker/docker/client"
	"github.com/elecbug/linuxus/src/ctl/internal/config"
)

type App struct {
	DockerClient *client.Client

	CurrentDir string
	ExecPath   string
	RepoDir    string
	SourceDir  string
	ConfigFile string

	Config  config.Config
	UserIDs []string
	SafeIDs []string
	Seen    map[string]struct{}
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

func (a *App) authImageName() string {
	return a.Config.AuthService.Container.Name + ":runtime"
}

func (a *App) userImageName() string {
	return a.Config.UserService.Container.NamePrefix + "base:runtime"
}

func (a *App) homeDirForUser(userID string) string {
	return filepath.Join(a.Config.Volumes.Host.Homes, userID)
}
