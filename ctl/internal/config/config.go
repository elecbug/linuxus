package config

type Config struct {
	UserService struct {
		SourceDir string `yaml:"source_dir"`

		Container struct {
			NamePrefix    string `yaml:"name_prefix"`
			NetworkPrefix string `yaml:"network_prefix"`
			BaseIP        string `yaml:"base_ip"`

			Runtime struct {
				UID      int    `yaml:"uid"`
				GID      int    `yaml:"gid"`
				User     string `yaml:"user"`
				Hostname string `yaml:"hostname"`
				Timezone string `yaml:"timezone"`
			} `yaml:"runtime"`

			User struct {
				Limits Limits `yaml:"limits"`
			} `yaml:"user"`

			Admin struct {
				UserID string `yaml:"user_id"`
				Limits Limits `yaml:"limits"`
			} `yaml:"admin"`
		} `yaml:"container"`
	} `yaml:"user_service"`

	AuthService struct {
		SourceDir string `yaml:"source_dir"`

		Container struct {
			Name         string `yaml:"name"`
			Timezone     string `yaml:"timezone"`
			ExternalPort int    `yaml:"external_port"`
		} `yaml:"container"`

		URLPath struct {
			Login    string `yaml:"login"`
			Logout   string `yaml:"logout"`
			Service  string `yaml:"service"`
			Terminal string `yaml:"terminal"`
		} `yaml:"url_path"`

		AuthListFile struct {
			HostPath      string `yaml:"host_path"`
			ContainerPath string `yaml:"container_path"`
		} `yaml:"auth_list_file"`

		Security struct {
			SessionSecret  string `yaml:"session_secret"`
			TrustedProxies string `yaml:"trusted_proxies"`
		} `yaml:"security"`
	} `yaml:"auth_service"`

	ManagerService struct {
		SourceDir string `yaml:"source_dir"`

		// Container defines runtime and network settings for manager
		Container struct {
			Name     string `yaml:"name"`
			Timezone string `yaml:"timezone"`
			Network  string `yaml:"network"`
			Subnet   string `yaml:"subnet"`
		} `yaml:"container"`

		// User defines session timeout settings for user sessions managed by the manager service.
		User struct {
			// Timeout is the duration a user session remains active without activity.
			Timeout string `yaml:"timeout"`
		} `yaml:"user"`

		// Session defines manager request/session timing behavior.
		Session struct {
			Timeout string `yaml:"timeout"`
		} `yaml:"session"`

		// Security defines authentication settings for manager endpoints.
		Security struct {
			ManagerSecret string `yaml:"manager_secret"`
		} `yaml:"security"`
	} `yaml:"manager_service"`

	Volumes struct {
		Host struct {
			Volumes  string `yaml:"volumes"`
			Homes    string `yaml:"homes"`
			Share    string `yaml:"share"`
			Readonly string `yaml:"readonly"`
		} `yaml:"host"`

		Container struct {
			Share    string `yaml:"share"`
			Readonly string `yaml:"readonly"`
		} `yaml:"container"`

		DiskLimit int `yaml:"disk_limit"` // MB
	} `yaml:"volumes"`
}

type Limits struct {
	CPU     any    `yaml:"cpu"`
	Memory  string `yaml:"memory"`
	PID     int    `yaml:"pid"`
	Disk    int    `yaml:"disk"`
	Ulimits struct {
		Nofile struct {
			Soft int `yaml:"soft"`
			Hard int `yaml:"hard"`
		} `yaml:"nofile"`
	} `yaml:"ulimits"`
}
