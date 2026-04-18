package spec

type ContainerInfo struct {
	Name   string
	Status string
	Image  string
	Ports  string
	UserID string
}

type NetworkInfo struct {
	Name   string
	ID     string
	Subnet string
}
