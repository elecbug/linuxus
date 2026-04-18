package spec

// ContainerLimits defines runtime limits applied to a container.
type ContainerLimits struct {
	// Memory is the memory limit string accepted by parser.
	Memory string
	// CPUs is the CPU limit string accepted by parser.
	CPUs string
	// Pids is the process count limit.
	Pids int
	// NofileSoft is the soft nofile ulimit.
	NofileSoft int
	// NofileHard is the hard nofile ulimit.
	NofileHard int
}

// RuntimeContainerSpec describes a Docker container to create and run.
type RuntimeContainerSpec struct {
	// Image is the image reference to run.
	Image string
	// Name is the target container name.
	Name string
	// Hostname is the hostname configured in the container.
	Hostname string
	// WorkingDir is the default working directory in container.
	WorkingDir string
	// User is the process user in container.
	User string
	// ReadOnly enables read-only root filesystem.
	ReadOnly bool
	// Tmpfs defines tmpfs mounts and options.
	Tmpfs []string
	// Environment contains environment variables.
	Environment []string
	// Volumes contains bind mount definitions.
	Volumes []string
	// Ports contains host:container mappings.
	Ports []string
	// Restart is the Docker restart policy.
	Restart string
	// SecurityOpt contains Docker security options.
	SecurityOpt []string
	// CapDrop contains Linux capabilities to drop.
	CapDrop []string
	// Limits contains CPU/memory/pid/ulimit constraints.
	Limits ContainerLimits
	// Networks is the list of networks to connect.
	Networks []string
}

// RuntimeNetworkSpec describes a Docker network to create.
type RuntimeNetworkSpec struct {
	// Name is the target network name.
	Name string
	// Subnet is the subnet CIDR for the network.
	Subnet string
}
