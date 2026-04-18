package spec

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
