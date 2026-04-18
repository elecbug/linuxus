package config

import "time"

// ResourceLimits defines Docker resource constraints for a runtime container.
type ResourceLimits struct {
	// NanoCPUs is the CPU limit in Docker NanoCPU units.
	NanoCPUs int64
	// MemoryBytes is the memory limit in bytes.
	MemoryBytes int64
	// PidsLimit is the process count limit.
	PidsLimit int64
	// NofileSoft is the soft nofile ulimit.
	NofileSoft int64
	// NofileHard is the hard nofile ulimit.
	NofileHard int64
}

// Config contains all runtime settings for the manager service.
type Config struct {
	// ListenAddr is the HTTP server bind address.
	ListenAddr string
	// UserImage is the Docker image used for user runtimes.
	UserImage string
	// UserContainerNamePrefix is prefixed to generated user container names.
	UserContainerNamePrefix string
	// NetworkPrefix is prefixed to generated user network names.
	NetworkPrefix string
	// BaseIP is the base IPv4 address used for subnet allocation.
	BaseIP string
	// AuthContainerName is the auth container to connect to user networks.
	AuthContainerName string
	// AdminUserID is the privileged user ID.
	AdminUserID string
	// ManagerSecret is an optional shared secret for protected endpoints.
	ManagerSecret string

	// RuntimeUser is the user identity used to run user containers.
	RuntimeUser string
	// ContainerRuntimeUser is the username expected inside user containers.
	ContainerRuntimeUser string
	// ContainerHostname is the hostname assigned to user containers.
	ContainerHostname string
	// WorkingDir is the working directory inside user containers.
	WorkingDir string
	// Timezone is the timezone value injected into user containers.
	Timezone string
	// ReadOnlyRootFS enables read-only root filesystem for user containers.
	ReadOnlyRootFS bool
	// ManagerWaitTime is the timeout for manager user-up operations.
	ManagerWaitTime time.Duration
	// ContainerTimeout is the idle timeout before cleanup.
	ContainerTimeout time.Duration

	// HostHomesDir is the host base path for per-user home directories.
	HostHomesDir string
	// HostShareDir is the host shared writable directory path.
	HostShareDir string
	// HostReadonlyDir is the host shared read-only directory path.
	HostReadonlyDir string
	// ContainerShareDir is the in-container mount point for writable shared data.
	ContainerShareDir string
	// ContainerReadonlyDir is the in-container mount point for read-only shared data.
	ContainerReadonlyDir string

	// UserLimits are runtime limits for regular users.
	UserLimits ResourceLimits
	// AdminLimits are runtime limits for admin user runtime containers.
	AdminLimits ResourceLimits
}
