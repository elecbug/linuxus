package spec

// ContainerInfo is a display model for runtime container status output.
type ContainerInfo struct {
	// Name is the container name.
	Name string
	// Status is the summarized runtime state text.
	Status string
	// Image is the container image name.
	Image string
	// Ports is the summarized port mapping text.
	Ports string
	// Role is the associated logical user identifier.
	Role string
}

// NetworkInfo is a display model for runtime network status output.
type NetworkInfo struct {
	// Name is the network name.
	Name string
	// ID is the shortened network ID.
	ID string
	// Subnet is the network subnet CIDR.
	Subnet string
}
