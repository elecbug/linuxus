package config

// Config defines the runtime settings and .env key components consumed by linuxusctl.
type Config struct {
	// UserService configures user image and runtime settings.
	UserService struct {
		// Container groups naming, runtime, and user limit settings.
		Container struct {
			// NamePrefix is prefixed to generated user container names.
			NamePrefix string `env:"NAME_PREFIX"`
			// NetworkNamePrefix is prefixed to generated user network names.
			NetworkNamePrefix string `env:"NETWORK_NAME_PREFIX"`
			// BaseSubnet16 is the base subnet used for user networking.
			BaseSubnet16 string `env:"BASE_SUBNET_16"`
		} `env:"CONTAINER"`

		// Runtime defines execution identity inside user containers.
		Runtime struct {
			// UID is the runtime user ID.
			UID int `env:"UID"`
			// GID is the runtime group ID.
			GID int `env:"GID"`
			// LinuxUsername is the runtime username.
			LinuxUsername string `env:"LINUX_USERNAME"`
			// LinuxHostname is the default container hostname.
			LinuxHostname string `env:"LINUX_HOSTNAME"`
			// Timezone is the timezone inside the container.
			Timezone string `env:"TIMEZONE"`
		} `env:"RUNTIME"`

		Limits struct {
			// User contains resource limits for user containers.
			User Limits `env:"USER"`
			// Admin contains resource limits for the admin user.
			Admin Limits `env:"ADMIN"`
		} `env:"LIMITS"`
	} `env:"USER_SERVICE"`

	// AuthService configures the authentication gateway service.
	AuthService struct {
		// Container defines auth container runtime settings.
		Container struct {
			// Name is the auth container name.
			Name string `env:"NAME"`
			// ExternalPort is the host port exposed by auth service.
			ExternalPort int `env:"EXTERNAL_PORT"`
		} `env:"CONTAINER"`

		// Runtime defines execution identity and timezone inside auth container.
		Runtime struct {
			// Timezone is the timezone inside the auth container.
			Timezone string `env:"TIMEZONE"`
		} `env:"RUNTIME"`

		// ServiceURL defines auth endpoint paths.
		ServiceURL struct {
			// Login is the login route path.
			Login string `env:"LOGIN"`
			// Logout is the logout route path.
			Logout string `env:"LOGOUT"`
			// Service is the base service route path.
			Service string `env:"SERVICE"`
			// Terminal is the terminal route path.
			Terminal string `env:"TERMINAL"`
			// Signup is the user registration route path.
			Signup string `env:"SIGNUP"`
		} `env:"SERVICE_URL"`

		// Mounts defines host/container paths for auth list data.
		Mounts struct {
			// HostAuthListPath is the auth list path on the host.
			HostAuthListPath string `env:"HOST_AUTH_LIST_PATH"`
			// ContainerAuthListPath is the mounted auth list path in container.
			ContainerAuthListPath string `env:"CONTAINER_AUTH_LIST_PATH"`
		} `env:"MOUNTS"`

		// Security defines auth security settings.
		Security struct {
			// SessionSecret signs auth session data.
			SessionSecret string `env:"SESSION_SECRET"`
			// TrustedProxies is the trusted proxy CIDR list.
			TrustedProxies string `env:"TRUSTED_PROXIES"`
		} `env:"SECURITY"`

		// AllowSignup enables or disables user self-registration.
		AllowSignup bool `env:"ALLOW_SIGNUP"`
	} `env:"AUTH_SERVICE"`

	// ManagerService configures manager runtime and session behavior.
	ManagerService struct {
		// Container defines runtime and network settings for manager.
		Container struct {
			// Name is the manager container name.
			Name string `env:"NAME"`
			// Network is the primary manager runtime network name.
			Network string `env:"NETWORK"`
			// Subnet is the subnet CIDR for manager runtime network.
			Subnet string `env:"SUBNET"`
			// HomesDir is the in-container mount point for user home directories.
			HomesDir string `env:"HOMES_DIR"`
			// ShareDir is the in-container mount point for writable shared data.
			ShareDir string `env:"SHARE_DIR"`
			// ReadonlyDir is the in-container mount point for read-only shared data.
			ReadonlyDir string `env:"READONLY_DIR"`
		} `env:"CONTAINER"`

		// UserManagement defines session timeout settings for managed users.
		UserManagement struct {
			// CleanupTimeout is the idle timeout for user sessions.
			CleanupTimeout string `env:"CLEANUP_TIMEOUT"`
		} `env:"USER_MANAGEMENT"`

		// AuthService defines manager request/session timing behavior.
		AuthService struct {
			// ConnectionTimeout is the manager request/session timeout duration.
			ConnectionTimeout string `env:"CONNECTION_TIMEOUT"`
		} `env:"AUTH_SERVICE"`

		// Security defines authentication settings for manager endpoints.
		Security struct {
			// SessionSecret signs manager session data.
			SessionSecret string `env:"SESSION_SECRET"`
		} `env:"SECURITY"`

		// AdminID is the admin user ID.
		AdminID string `env:"ADMIN_ID"`
	} `env:"MANAGER_SERVICE"`

	// Volumes configures host/container volume paths and default disk size.
	Volumes struct {
		// AutoEnsure prepares user disks on demand through the host disk service.
		AutoEnsure bool `env:"AUTO_ENSURE"`

		// Host contains host-side directories.
		Host struct {
			// Volumes is the root host directory for managed volume data.
			Volumes string `env:"VOLUMES"`
			// Homes is the host path for per-user home disks.
			Homes string `env:"HOMES"`
			// Share is the host path for shared writable data.
			Share string `env:"SHARE"`
			// Readonly is the host path for shared read-only data.
			Readonly string `env:"READONLY"`
		} `env:"HOST"`

		// Container contains container-side mount points.
		Container struct {
			// Share is the writable shared mount path in containers.
			Share string `env:"SHARE"`
			// Readonly is the read-only shared mount path in containers.
			Readonly string `env:"READONLY"`
		} `env:"CONTAINER"`

		// DiskLimit is the default disk image size string (e.g., 1G, 512M).
		DiskLimit string `env:"DISK_LIMIT"`
	} `env:"VOLUMES"`
}

// Limits defines per-container resource limits from configuration.
type Limits struct {
	// CPU is the CPU limit value (e.g., 0.5, 1, 2).
	CPU any `env:"CPU"`
	// Memory is the memory limit string (e.g., 512m, 1g).
	Memory string `env:"MEMORY"`
	// PID is the process count limit.
	PID int `env:"PID"`
	// Disk is the per-user disk size string (e.g., 1G, 512M).
	Disk string `env:"DISK"`
	// Ulimits contains configurable Unix resource limits.
	Ulimits struct {
		Nofile struct {
			// Soft is the soft open-file descriptor limit.
			Soft int `env:"SOFT"`
			// Hard is the hard open-file descriptor limit.
			Hard int `env:"HARD"`
		} `env:"NOFILE"`
	} `env:"ULIMITS"`
}
