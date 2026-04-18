package config

import "time"

// ResourceLimits defines container-level runtime resource constraints.
type ResourceLimits struct {
	// NanoCPUs is the CPU quota in Docker NanoCPU units.
	NanoCPUs int64
	// MemoryBytes is the memory limit in bytes.
	MemoryBytes int64
	// PidsLimit is the maximum number of processes allowed.
	PidsLimit int64
	// NofileSoft is the soft file descriptor limit.
	NofileSoft int64
	// NofileHard is the hard file descriptor limit.
	NofileHard int64
}

// Config stores manager service runtime configuration.
type Config struct {
	// ListenAddr is the HTTP listen address for the manager API.
	ListenAddr string
	// UserImage is the image used for per-user runtime containers.
	UserImage string
	// UserContainerNamePrefix prefixes runtime container names.
	UserContainerNamePrefix string
	// NetworkPrefix prefixes runtime network names.
	NetworkPrefix string
	// BaseIP is the base IPv4 address used for subnet allocation.
	BaseIP string
	// AuthContainerName is the shared auth container name.
	AuthContainerName string
	// AdminUserID identifies the admin user.
	AdminUserID string

	// RuntimeUser is the uid:gid (or user) applied to the runtime process.
	RuntimeUser string
	// ContainerRuntimeUser is the runtime username expected inside containers.
	ContainerRuntimeUser string
	// ContainerHostname is the hostname assigned to user containers.
	ContainerHostname string
	// WorkingDir is the working directory for user container processes.
	WorkingDir string
	// Timezone is the timezone passed to containers.
	Timezone string
	// ReadOnlyRootFS enables read-only root filesystem mode.
	ReadOnlyRootFS bool
	// ManagerWaitTime is the timeout used for user runtime preparation requests.
	ManagerWaitTime time.Duration
	// Timeout is the duration the container remains alive from the user's last activity time.
	ContainerTimeout time.Duration

	// HostHomesDir is the host directory root for user home mounts.
	HostHomesDir string
	// HostShareDir is the shared writable host directory.
	HostShareDir string
	// HostReadonlyDir is the shared readonly host directory.
	HostReadonlyDir string
	// ContainerShareDir is the shared writable mount path in containers.
	ContainerShareDir string
	// ContainerReadonlyDir is the shared readonly mount path in containers.
	ContainerReadonlyDir string

	// UserLimits applies to regular users.
	UserLimits ResourceLimits
	// AdminLimits applies to the admin user.
	AdminLimits ResourceLimits
}
