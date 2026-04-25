package config

// Config defines the full YAML schema consumed by linuxusctl.
type Config struct {
	// UserService configures user image and runtime settings.
	UserService struct {
		// SourceDir is the Docker build context path for user service.
		SourceDir string `yaml:"source_dir"`

		// Container groups naming, runtime, and user limit settings.
		Container struct {
			// NamePrefix is prefixed to generated user container names.
			NamePrefix string `yaml:"name_prefix"`
			// NetworkNamePrefix is prefixed to generated user network names.
			NetworkNamePrefix string `yaml:"network_name_prefix"`
			// BaseSubnet16 is the base subnet used for user networking.
			BaseSubnet16 string `yaml:"base_subnet_16"`
		} `yaml:"container"`

		// Runtime defines execution identity inside user containers.
		Runtime struct {
			// UID is the runtime user ID.
			UID int `yaml:"uid"`
			// GID is the runtime group ID.
			GID int `yaml:"gid"`
			// LinuxUsername is the runtime username.
			LinuxUsername string `yaml:"linux_username"`
			// LinuxHostname is the default container hostname.
			LinuxHostname string `yaml:"linux_hostname"`
			// Timezone is the timezone inside the container.
			Timezone string `yaml:"timezone"`
		} `yaml:"runtime"`

		Limits struct {
			// User contains resource limits for user containers.
			User Limits `yaml:"user"`
			// Admin contains resource limits for the admin user.
			Admin Limits `yaml:"admin"`
		} `yaml:"limits"`
	} `yaml:"user_service"`

	// AuthService configures the authentication gateway service.
	AuthService struct {
		// SourceDir is the Docker build context path for auth service.
		SourceDir string `yaml:"source_dir"`

		// Container defines auth container runtime settings.
		Container struct {
			// Name is the auth container name.
			Name string `yaml:"name"`
			// ExternalPort is the host port exposed by auth service.
			ExternalPort int `yaml:"external_port"`
		} `yaml:"container"`

		// Runtime defines execution identity and timezone inside auth container.
		Runtime struct {
			// Timezone is the timezone inside the auth container.
			Timezone string `yaml:"timezone"`
		} `yaml:"runtime"`

		// ServiceURL defines auth endpoint paths.
		ServiceURL struct {
			// Login is the login route path.
			Login string `yaml:"login"`
			// Logout is the logout route path.
			Logout string `yaml:"logout"`
			// Service is the base service route path.
			Service string `yaml:"service"`
			// Terminal is the terminal route path.
			Terminal string `yaml:"terminal"`
			// Signup is the user registration route path.
			Signup string `yaml:"signup"`
		} `yaml:"service_url"`

		// Mounts defines host/container paths for auth list data.
		Mounts struct {
			// HostAuthListPath is the auth list path on the host.
			HostAuthListPath string `yaml:"host_auth_list_path"`
			// ContainerAuthListPath is the mounted auth list path in container.
			ContainerAuthListPath string `yaml:"container_auth_list_path"`
		} `yaml:"mounts"`

		// Security defines auth security settings.
		Security struct {
			// SessionSecret signs auth session data.
			SessionSecret string `yaml:"session_secret"`
			// TrustedProxies is the trusted proxy CIDR list.
			TrustedProxies string `yaml:"trusted_proxies"`
		} `yaml:"security"`

		// AllowSignup enables or disables user self-registration.
		AllowSignup bool `yaml:"allow_signup"`
	} `yaml:"auth_service"`

	// ManagerService configures manager runtime and session behavior.
	ManagerService struct {
		// SourceDir is the Docker build context path for manager service.
		SourceDir string `yaml:"source_dir"`

		// Container defines runtime and network settings for manager.
		Container struct {
			// Name is the manager container name.
			Name string `yaml:"name"`
			// Network is the primary manager runtime network name.
			Network string `yaml:"network"`
			// Subnet is the subnet CIDR for manager runtime network.
			Subnet string `yaml:"subnet"`
			// HomesDir is the in-container mount point for user home directories.
			HomesDir string `yaml:"homes_dir"`
			// ShareDir is the in-container mount point for writable shared data.
			ShareDir string `yaml:"share_dir"`
			// ReadonlyDir is the in-container mount point for read-only shared data.
			ReadonlyDir string `yaml:"readonly_dir"`
		} `yaml:"container"`

		// UserManagement defines session timeout settings for managed users.
		UserManagement struct {
			// CleanupTimeout is the idle timeout for user sessions.
			CleanupTimeout string `yaml:"cleanup_timeout"`
		} `yaml:"user_management"`

		// AuthService defines manager request/session timing behavior.
		AuthService struct {
			// ConnectionTimeout is the manager request/session timeout duration.
			ConnectionTimeout string `yaml:"connection_timeout"`
		} `yaml:"auth_service"`

		// Security defines authentication settings for manager endpoints.
		Security struct {
			// SessionSecret signs manager session data.
			SessionSecret string `yaml:"session_secret"`
		} `yaml:"security"`

		// AdminID is the admin user ID.
		AdminID string `yaml:"admin_id"`
	} `yaml:"manager_service"`

	// Volumes configures host/container volume paths and default disk size.
	Volumes struct {
		// Host contains host-side directories.
		Host struct {
			// Volumes is the root host directory for managed volume data.
			Volumes string `yaml:"volumes"`
			// Homes is the host path for per-user home disks.
			Homes string `yaml:"homes"`
			// Share is the host path for shared writable data.
			Share string `yaml:"share"`
			// Readonly is the host path for shared read-only data.
			Readonly string `yaml:"readonly"`
		} `yaml:"host"`

		// Container contains container-side mount points.
		Container struct {
			// Share is the writable shared mount path in containers.
			Share string `yaml:"share"`
			// Readonly is the read-only shared mount path in containers.
			Readonly string `yaml:"readonly"`
		} `yaml:"container"`

		// DiskLimit is the default disk image size string (e.g., 1G, 512M).
		DiskLimit string `yaml:"disk_limit"`
	} `yaml:"volumes"`
}

// Limits defines per-container resource limits from configuration.
type Limits struct {
	// CPU is the CPU limit value (e.g., 0.5, 1, 2).
	CPU any `yaml:"cpu"`
	// Memory is the memory limit string (e.g., 512m, 1g).
	Memory string `yaml:"memory"`
	// PID is the process count limit.
	PID int `yaml:"pid"`
	// Disk is the per-user disk size string (e.g., 1G, 512M).
	Disk string `yaml:"disk"`
	// Ulimits contains configurable Unix resource limits.
	Ulimits struct {
		Nofile struct {
			// Soft is the soft open-file descriptor limit.
			Soft int `yaml:"soft"`
			// Hard is the hard open-file descriptor limit.
			Hard int `yaml:"hard"`
		} `yaml:"nofile"`
	} `yaml:"ulimits"`
}
