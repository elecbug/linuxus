package app

import (
	"github.com/elecbug/linuxus/src/ctl/internal/config"
)

type App struct {
	CurrentDir string
	ExecPath   string
	UtilDir    string
	RepoDir    string
	SourceDir  string
	ConfigFile string

	Config  config.Config
	UserIDs []string
	SafeIDs []string
	Seen    map[string]struct{}
}

type ComposeFile struct {
	Version  string                    `yaml:"version"`
	Services map[string]ComposeService `yaml:"services"`
	Networks map[string]ComposeNetwork `yaml:"networks"`
}

type ComposeService struct {
	User        string                 `yaml:"user,omitempty"`
	Build       *ComposeBuild          `yaml:"build,omitempty"`
	Container   string                 `yaml:"container_name,omitempty"`
	Hostname    string                 `yaml:"hostname,omitempty"`
	WorkingDir  string                 `yaml:"working_dir,omitempty"`
	ReadOnly    bool                   `yaml:"read_only,omitempty"`
	Tmpfs       []string               `yaml:"tmpfs,omitempty"`
	Environment []string               `yaml:"environment,omitempty"`
	Volumes     []string               `yaml:"volumes,omitempty"`
	Ports       []string               `yaml:"ports,omitempty"`
	Restart     string                 `yaml:"restart,omitempty"`
	SecurityOpt []string               `yaml:"security_opt,omitempty"`
	CapDrop     []string               `yaml:"cap_drop,omitempty"`
	MemLimit    string                 `yaml:"mem_limit,omitempty"`
	CPUs        string                 `yaml:"cpus,omitempty"`
	PidsLimit   int                    `yaml:"pids_limit,omitempty"`
	Ulimits     map[string]NofileLimit `yaml:"ulimits,omitempty"`
	Networks    []string               `yaml:"networks,omitempty"`
}

type ComposeBuild struct {
	Context string   `yaml:"context"`
	Args    []string `yaml:"args,omitempty"`
}

type NofileLimit struct {
	Soft int `yaml:"soft"`
	Hard int `yaml:"hard"`
}

type ComposeNetwork struct {
	Driver string       `yaml:"driver"`
	IPAM   *ComposeIPAM `yaml:"ipam,omitempty"`
}

type ComposeIPAM struct {
	Config []ComposeSubnet `yaml:"config"`
}

type ComposeSubnet struct {
	Subnet string `yaml:"subnet"`
}
