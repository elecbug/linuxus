package config

import "time"

type ResourceLimits struct {
	NanoCPUs    int64
	MemoryBytes int64
	PidsLimit   int64
	NofileSoft  int64
	NofileHard  int64
}

type Config struct {
	ListenAddr              string
	UserImage               string
	UserContainerNamePrefix string
	NetworkPrefix           string
	BaseIP                  string
	AuthContainerName       string
	AdminUserID             string
	ManagerSecret           string

	RuntimeUser          string
	ContainerRuntimeUser string
	ContainerHostname    string
	WorkingDir           string
	Timezone             string
	ReadOnlyRootFS       bool
	ManagerWaitTime      time.Duration
	ContainerTimeout     time.Duration

	HostHomesDir         string
	HostShareDir         string
	HostReadonlyDir      string
	ContainerShareDir    string
	ContainerReadonlyDir string

	UserLimits  ResourceLimits
	AdminLimits ResourceLimits
}
