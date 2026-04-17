package config

// Config is the root YAML configuration consumed by ctl runtime operations.
type Config struct {
	// UserService defines source and container settings for per-user runtime workloads.
	UserService struct {
		// SourceDir is the path to the user service build context.
		SourceDir string `yaml:"source_dir"`

		// Container groups container-level settings for user runtime services.
		Container struct {
			// NamePrefix is the prefix used when naming user containers.
			NamePrefix string `yaml:"name_prefix"`
			// NetworkPrefix is the prefix used when naming user networks.
			NetworkPrefix string `yaml:"network_prefix"`
			// BaseIP is the base address used for runtime network allocation.
			BaseIP string `yaml:"base_ip"`

			// Runtime defines runtime identity and environment defaults inside user containers.
			Runtime struct {
				// UID is the Linux user ID applied to runtime files and mounts.
				UID int `yaml:"uid"`
				// GID is the Linux group ID applied to runtime files and mounts.
				GID int `yaml:"gid"`
				// User is the runtime username used inside containers.
				User string `yaml:"user"`
				// Hostname is the default hostname used for runtime containers.
				Hostname string `yaml:"hostname"`
				// Timezone is the timezone passed to runtime containers.
				Timezone string `yaml:"timezone"`
			} `yaml:"runtime"`

			// User holds resource limits for regular user containers.
			User struct {
				// Limits defines CPU, memory, process, file, and disk limits.
				Limits Limits `yaml:"limits"`
			} `yaml:"user"`

			// Admin holds identity and limits for the admin user container.
			Admin struct {
				// UserID identifies the admin account in the auth list.
				UserID string `yaml:"user_id"`
				// Limits defines CPU, memory, process, file, and disk limits.
				Limits Limits `yaml:"limits"`
			} `yaml:"admin"`
		} `yaml:"container"`
	} `yaml:"user_service"`

	// AuthService defines source, network, and auth-path settings for the auth gateway.
	AuthService struct {
		// SourceDir is the path to the auth service build context.
		SourceDir string `yaml:"source_dir"`
		// Container defines runtime and exposure settings for the auth service.
		Container struct {
			// Name is the auth container name.
			Name string `yaml:"name"`
			// Timezone is the timezone passed to the auth container.
			Timezone string `yaml:"timezone"`
			// ExternalPort is the host port mapped to the auth service.
			ExternalPort int `yaml:"external_port"`
		} `yaml:"container"`

		// URLPath defines auth service endpoint paths.
		URLPath struct {
			// Login is the path for login endpoint.
			Login string `yaml:"login"`
			// Logout is the path for logout endpoint.
			Logout string `yaml:"logout"`
			// Service is the path for service callback endpoint.
			Service string `yaml:"service"`
			// Terminal is the path for terminal access endpoint.
			Terminal string `yaml:"terminal"`
		} `yaml:"url_path"`

		// AuthListFile defines host and container locations for the auth list file.
		AuthListFile struct {
			// HostPath is the auth list path on the host filesystem.
			HostPath string `yaml:"host_path"`
			// ContainerPath is the auth list path inside the auth container.
			ContainerPath string `yaml:"container_path"`
		} `yaml:"auth_list_file"`

		// Security defines secrets and trusted proxy settings for auth.
		Security struct {
			// SessionSecret signs or encrypts auth session state.
			SessionSecret string `yaml:"session_secret"`
			// TrustedProxies lists proxy ranges trusted for forwarded headers.
			TrustedProxies string `yaml:"trusted_proxies"`
		} `yaml:"security"`
	} `yaml:"auth_service"`

	// ManagerService defines source and container settings for the manager service.
	ManagerService struct {
		// SourceDir is the path to the manager service build context.
		SourceDir string `yaml:"source_dir"`
		// Container defines runtime and network settings for manager.
		Container struct {
			// Name is the manager container name.
			Name string `yaml:"name"`
			// Timezone is the timezone passed to the manager container.
			Timezone string `yaml:"timezone"`
			// Network is the dedicated Docker network for manager-managed services.
			Network string `yaml:"network"`
			// Subnet is the CIDR subnet assigned to the manager network.
			Subnet string `yaml:"subnet"`
		} `yaml:"container"`
		// Session defines manager request/session timing behavior.
		Session struct {
			// Timeout is the duration used for manager-side wait and session windows.
			Timeout string `yaml:"timeout"`
		} `yaml:"session"`
	} `yaml:"manager_service"`

	// Volumes defines host/container mount points and disk size defaults.
	Volumes struct {
		// Host defines host-side volume roots used by runtime services.
		Host struct {
			// Volumes is the host root directory for runtime-managed volume data.
			Volumes string `yaml:"volumes"`
			// Homes is the host directory containing per-user home mounts.
			Homes string `yaml:"homes"`
			// Share is the host path mounted as shared writable storage.
			Share string `yaml:"share"`
			// Readonly is the host path mounted as shared read-only storage.
			Readonly string `yaml:"readonly"`
		} `yaml:"host"`
		// Container defines mount points as seen from inside containers.
		Container struct {
			// Share is the shared writable mount path inside containers.
			Share string `yaml:"share"`
			// Readonly is the shared read-only mount path inside containers.
			Readonly string `yaml:"readonly"`
		} `yaml:"container"`
		// DiskLimit is the default disk image size in MB.
		DiskLimit int `yaml:"disk_limit"` // MB
	} `yaml:"volumes"`
}

// Limits defines runtime resource limits for a container profile.
type Limits struct {
	// CPU is the CPU quota value accepted from configuration.
	CPU any `yaml:"cpu"`
	// Memory is the memory limit string (for example, 512m or 1g).
	Memory string `yaml:"memory"`
	// PID is the max number of processes allowed.
	PID int `yaml:"pid"`
	// Disk is the disk image size in MB for user home storage.
	Disk int `yaml:"disk"`
	// Ulimits groups Linux ulimit values to apply.
	Ulimits struct {
		// Nofile defines soft/hard limits for open file descriptors.
		Nofile struct {
			// Soft is the soft nofile limit.
			Soft int `yaml:"soft"`
			// Hard is the hard nofile limit.
			Hard int `yaml:"hard"`
		} `yaml:"nofile"`
	} `yaml:"ulimits"`
}
