package config

import "time"

type Config struct {
	ListenAddr              string
	UserImage               string
	UserContainerNamePrefix string
	NetworkPrefix           string
	BaseIP                  string
	AuthContainerName       string
	AdminUserID             string

	RuntimeUser          string
	ContainerRuntimeUser string
	ContainerHostname    string
	WorkingDir           string
	Timezone             string
	ReadOnlyRootFS       bool
	ManagerWaitTime      time.Duration

	HostHomesDir         string
	HostShareDir         string
	HostReadonlyDir      string
	ContainerShareDir    string
	ContainerReadonlyDir string
}
