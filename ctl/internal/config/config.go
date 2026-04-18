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
			// NetworkPrefix is prefixed to generated user network names.
			NetworkPrefix string `yaml:"network_prefix"`
			// BaseIP is the base IP range used for user networking.
			BaseIP string `yaml:"base_ip"`

			// Runtime defines execution identity inside user containers.
			Runtime struct {
				// UID is the runtime user ID.
				UID int `yaml:"uid"`
				// GID is the runtime group ID.
				GID int `yaml:"gid"`
				// User is the runtime username.
				User string `yaml:"user"`
				// Hostname is the default container hostname.
				Hostname string `yaml:"hostname"`
				// Timezone is the timezone inside the container.
				Timezone string `yaml:"timezone"`
			} `yaml:"runtime"`

			// User stores limits for regular users.
			User struct {
				// Limits are resource limits for regular users.
				Limits Limits `yaml:"limits"`
			} `yaml:"user"`

			// Admin stores identity and limits for the admin user.
			Admin struct {
				// UserID is the admin user ID.
				UserID string `yaml:"user_id"`
				// Limits are resource limits for the admin user.
				Limits Limits `yaml:"limits"`
			} `yaml:"admin"`
		} `yaml:"container"`
	} `yaml:"user_service"`

	// AuthService configures the authentication gateway service.
	AuthService struct {
		// SourceDir is the Docker build context path for auth service.
		SourceDir string `yaml:"source_dir"`

		// Container defines auth container runtime settings.
		Container struct {
			// Name is the auth container name.
			Name string `yaml:"name"`
			// Timezone is the timezone inside the auth container.
			Timezone string `yaml:"timezone"`
			// ExternalPort is the host port exposed by auth service.
			ExternalPort int `yaml:"external_port"`
		} `yaml:"container"`

		// URLPath defines auth endpoint paths.
		URLPath struct {
			// Login is the login route path.
			Login string `yaml:"login"`
			// Logout is the logout route path.
			Logout string `yaml:"logout"`
			// Service is the base service route path.
			Service string `yaml:"service"`
			// Terminal is the terminal route path.
			Terminal string `yaml:"terminal"`
		} `yaml:"url_path"`

		// AuthListFile defines host/container paths for auth list data.
		AuthListFile struct {
			// HostPath is the auth list path on the host.
			HostPath string `yaml:"host_path"`
			// ContainerPath is the mounted auth list path in container.
			ContainerPath string `yaml:"container_path"`
		} `yaml:"auth_list_file"`

		// Security defines auth security settings.
		Security struct {
			// SessionSecret signs auth session data.
			SessionSecret string `yaml:"session_secret"`
			// TrustedProxies is the trusted proxy CIDR list.
			TrustedProxies string `yaml:"trusted_proxies"`
		} `yaml:"security"`
	} `yaml:"auth_service"`

	// ManagerService configures manager runtime and session behavior.
	ManagerService struct {
		// SourceDir is the Docker build context path for manager service.
		SourceDir string `yaml:"source_dir"`

		// Container defines runtime and network settings for manager.
		Container struct {
			// Name is the manager container name.
			Name string `yaml:"name"`
			// Timezone is the timezone inside manager container.
			Timezone string `yaml:"timezone"`
			// Network is the primary manager runtime network name.
			Network string `yaml:"network"`
			// Subnet is the subnet CIDR for manager runtime network.
			Subnet string `yaml:"subnet"`
		} `yaml:"container"`

		// User defines session timeout settings for managed users.
		User struct {
			// Timeout is the idle timeout for user sessions.
			Timeout string `yaml:"timeout"`
		} `yaml:"user"`

		// Session defines manager request/session timing behavior.
		Session struct {
			// Timeout is the manager request/session timeout duration.
			Timeout string `yaml:"timeout"`
		} `yaml:"session"`

		// Security defines authentication settings for manager endpoints.
		Security struct {
			// ManagerSecret authenticates privileged manager operations.
			ManagerSecret string `yaml:"manager_secret"`
		} `yaml:"security"`
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

		// DiskLimit is the default disk image size in MB.
		DiskLimit int `yaml:"disk_limit"`
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
	// Disk is the per-user disk size in MB.
	Disk int `yaml:"disk"`
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
