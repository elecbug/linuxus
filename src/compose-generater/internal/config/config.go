package config

type Config struct {
	ContainerRuntime struct {
		UID      int    `yaml:"uid"`
		GID      int    `yaml:"gid"`
		User     string `yaml:"user"`
		Hostname string `yaml:"hostname"`
		Timezone string `yaml:"timezone"`
	} `yaml:"container_runtime"`

	UserService struct {
		SourceDir           string `yaml:"source_dir"`
		ContainerNamePrefix string `yaml:"container_name_prefix"`
		NetworkPrefix       string `yaml:"network_prefix"`
		BaseIP              string `yaml:"base_ip"`
	} `yaml:"user_service"`

	Admin struct {
		UserID string `yaml:"user_id"`
	} `yaml:"admin"`

	UserLimits  Limits `yaml:"user_limits"`
	AdminLimits Limits `yaml:"admin_limits"`

	Volumes struct {
		Host struct {
			Base     string `yaml:"base"`
			Homes    string `yaml:"homes"`
			Share    string `yaml:"share"`
			Readonly string `yaml:"readonly"`
		} `yaml:"host"`
		Container struct {
			Share    string `yaml:"share"`
			Readonly string `yaml:"readonly"`
		} `yaml:"container"`
	} `yaml:"volumes"`

	URLPaths struct {
		Login    string `yaml:"login"`
		Logout   string `yaml:"logout"`
		Service  string `yaml:"service"`
		Terminal string `yaml:"terminal"`
	} `yaml:"url_paths"`

	AuthService struct {
		ExternalPort   int    `yaml:"external_port"`
		Timezone       string `yaml:"timezone"`
		SourceDir      string `yaml:"source_dir"`
		ContainerName  string `yaml:"container_name"`
		ListFile       string `yaml:"list_file"`
		ListMountPath  string `yaml:"list_mount_path"`
		SessionSecret  string `yaml:"session_secret"`
		TrustedProxies string `yaml:"trusted_proxies"`
	} `yaml:"auth_service"`

	Compose struct {
		OutputFile string `yaml:"output_file"`
	} `yaml:"compose"`
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
