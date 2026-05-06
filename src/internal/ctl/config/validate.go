package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/elecbug/linuxus/src/internal/common/convert"
	"github.com/elecbug/linuxus/src/internal/common/ruleset"
	"github.com/elecbug/linuxus/src/internal/common/subnet"
)

// ValidateConfig validates required config values before runtime operations.
func ValidateConfig(cfg *Config) error {
	errMsgs := []string{}

	if cfg.UserService.Container.NamePrefix == "" {
		errMsgs = append(errMsgs, "user_service.container.name_prefix is required")
	} else if !ruleset.AllowedDockerPrefix(cfg.UserService.Container.NamePrefix) {
		errMsgs = append(errMsgs, "user_service.container.name_prefix must be a valid Docker prefix")
	}

	if cfg.UserService.Container.NetworkNamePrefix == "" {
		errMsgs = append(errMsgs, "user_service.container.network_name_prefix is required")
	} else if !ruleset.AllowedDockerPrefix(cfg.UserService.Container.NetworkNamePrefix) {
		errMsgs = append(errMsgs, "user_service.container.network_name_prefix must be a valid Docker prefix")
	}

	if cfg.UserService.Container.BaseSubnet16 == "" {
		errMsgs = append(errMsgs, "user_service.container.base_subnet_16 is required")
	} else if !subnet.IsValidSubnet16(cfg.UserService.Container.BaseSubnet16) {
		errMsgs = append(errMsgs, "user_service.container.base_subnet_16 must be a valid /16 subnet (x.x.0.0)")
	}

	if cfg.UserService.Runtime.UID == 0 {
		errMsgs = append(errMsgs, "user_service.runtime.uid is required and must be non-zero")
	}

	if cfg.UserService.Runtime.GID == 0 {
		errMsgs = append(errMsgs, "user_service.runtime.gid is required and must be non-zero")
	}

	if cfg.UserService.Runtime.LinuxUsername == "" {
		errMsgs = append(errMsgs, "user_service.runtime.linux_username is required")
	} else if cfg.UserService.Runtime.LinuxUsername == "root" {
		errMsgs = append(errMsgs, "user_service.runtime.linux_username cannot be 'root'")
	} else if !ruleset.AllowedDockerID(cfg.UserService.Runtime.LinuxUsername) {
		errMsgs = append(errMsgs, "user_service.runtime.linux_username must be a valid Docker ID")
	}

	if cfg.UserService.Runtime.LinuxHostname == "" {
		errMsgs = append(errMsgs, "user_service.runtime.linux_hostname is required")
	} else if !ruleset.AllowedDockerID(cfg.UserService.Runtime.LinuxHostname) {
		errMsgs = append(errMsgs, "user_service.runtime.linux_hostname must be a valid Docker ID")
	}

	if cfg.UserService.Runtime.Timezone == "" {
		errMsgs = append(errMsgs, "user_service.runtime.timezone is required")
	}

	if err := validateLimits(cfg.UserService.Limits.User); err != nil {
		errMsgs = append(errMsgs, fmt.Sprintf("user_service.limits.user (%v)", err))
	}

	if err := validateLimits(cfg.UserService.Limits.Admin); err != nil {
		errMsgs = append(errMsgs, fmt.Sprintf("user_service.limits.admin (%v)", err))
	}

	if cfg.AuthService.Container.Name == "" {
		errMsgs = append(errMsgs, "auth_service.container.name is required")
	} else if !ruleset.AllowedDockerID(cfg.AuthService.Container.Name) {
		errMsgs = append(errMsgs, "auth_service.container.name must be a valid Docker ID")
	}

	if cfg.AuthService.Container.ExternalPort <= 0 || cfg.AuthService.Container.ExternalPort > 65535 {
		errMsgs = append(errMsgs, "auth_service.container.external_port must be a valid port number (1-65535)")
	}

	if cfg.AuthService.Runtime.Timezone == "" {
		errMsgs = append(errMsgs, "auth_service.runtime.timezone is required")
	}

	if cfg.AuthService.ServiceURL.Login == "" {
		errMsgs = append(errMsgs, "auth_service.service_url.login is required")
	}

	if cfg.AuthService.ServiceURL.Logout == "" {
		errMsgs = append(errMsgs, "auth_service.service_url.logout is required")
	}

	if cfg.AuthService.ServiceURL.Service == "" {
		errMsgs = append(errMsgs, "auth_service.service_url.service is required")
	}

	if cfg.AuthService.ServiceURL.Terminal == "" {
		errMsgs = append(errMsgs, "auth_service.service_url.terminal is required")
	}

	if cfg.AuthService.ServiceURL.Signup == "" {
		errMsgs = append(errMsgs, "auth_service.service_url.signup is required")
	}

	if cfg.AuthService.Mounts.HostAuthListPath == "" {
		errMsgs = append(errMsgs, "auth_service.mounts.host_auth_list_path is required")
	} else if err := UsablePath(cfg.AuthService.Mounts.HostAuthListPath); err != nil {
		errMsgs = append(errMsgs, fmt.Sprintf("auth_service.mounts.host_auth_list_path (%v)", err))
	}

	if cfg.AuthService.Mounts.ContainerAuthListPath == "" {
		errMsgs = append(errMsgs, "auth_service.mounts.container_auth_list_path is required")
	} else if !strings.HasPrefix(cfg.AuthService.Mounts.ContainerAuthListPath, "/") {
		errMsgs = append(errMsgs, "auth_service.mounts.container_auth_list_path must start with '/'")
	}

	if cfg.AuthService.Security.SessionSecret == "" {
		errMsgs = append(errMsgs, "auth_service.security.session_secret is required")
	}

	if err := subnet.IsValidSubnetList(cfg.AuthService.Security.TrustedProxies); err != nil {
		errMsgs = append(errMsgs, fmt.Sprintf("auth_service.security.trusted_proxies (%v)", err))
	}

	if cfg.ManagerService.Container.Name == "" {
		errMsgs = append(errMsgs, "manager_service.container.name is required")
	} else if !ruleset.AllowedDockerID(cfg.ManagerService.Container.Name) {
		errMsgs = append(errMsgs, "manager_service.container.name must be a valid Docker ID")
	} else if cfg.ManagerService.Container.Name == cfg.AuthService.Container.Name {
		errMsgs = append(errMsgs, "manager_service.container.name cannot be the same as auth_service.container.name")
	}

	if cfg.ManagerService.Container.Network == "" {
		errMsgs = append(errMsgs, "manager_service.container.network is required")
	} else if !ruleset.AllowedDockerID(cfg.ManagerService.Container.Network) {
		errMsgs = append(errMsgs, "manager_service.container.network must be a valid Docker ID")
	} else if cfg.ManagerService.Container.Network == cfg.UserService.Container.NetworkNamePrefix {
		errMsgs = append(errMsgs, "manager_service.container.network cannot be the same as user_service.container.network_name_prefix")
	}

	if cfg.ManagerService.Container.Subnet == "" {
		errMsgs = append(errMsgs, "manager_service.container.subnet is required")
	} else if !subnet.IsValidSubnet(cfg.ManagerService.Container.Subnet) {
		errMsgs = append(errMsgs, "manager_service.container.subnet must be a valid subnet")
	}

	if cfg.ManagerService.Container.HomesDir == "" {
		errMsgs = append(errMsgs, "manager_service.container.homes_dir is required")
	} else if !strings.HasPrefix(cfg.ManagerService.Container.HomesDir, "/") {
		errMsgs = append(errMsgs, "manager_service.container.homes_dir must start with '/'")
	}

	if cfg.ManagerService.Container.ShareDir == "" {
		errMsgs = append(errMsgs, "manager_service.container.share_dir is required")
	} else if !strings.HasPrefix(cfg.ManagerService.Container.ShareDir, "/") {
		errMsgs = append(errMsgs, "manager_service.container.share_dir must start with '/'")
	}

	if cfg.ManagerService.Container.ReadonlyDir == "" {
		errMsgs = append(errMsgs, "manager_service.container.readonly_dir is required")
	} else if !strings.HasPrefix(cfg.ManagerService.Container.ReadonlyDir, "/") {
		errMsgs = append(errMsgs, "manager_service.container.readonly_dir must start with '/'")
	}

	if cfg.ManagerService.Container.HomesDir != "" && cfg.ManagerService.Container.ShareDir != "" && cfg.ManagerService.Container.HomesDir == cfg.ManagerService.Container.ShareDir {
		errMsgs = append(errMsgs, "manager_service.container.homes_dir and manager_service.container.share_dir cannot be the same")
	}

	if cfg.ManagerService.Container.HomesDir != "" && cfg.ManagerService.Container.ReadonlyDir != "" && cfg.ManagerService.Container.HomesDir == cfg.ManagerService.Container.ReadonlyDir {
		errMsgs = append(errMsgs, "manager_service.container.homes_dir and manager_service.container.readonly_dir cannot be the same")
	}

	if cfg.ManagerService.Container.ShareDir != "" && cfg.ManagerService.Container.ReadonlyDir != "" && cfg.ManagerService.Container.ShareDir == cfg.ManagerService.Container.ReadonlyDir {
		errMsgs = append(errMsgs, "manager_service.container.share_dir and manager_service.container.readonly_dir cannot be the same")
	}

	if cfg.ManagerService.UserManagement.CleanupTimeout == "" {
		errMsgs = append(errMsgs, "manager_service.user_management.cleanup_timeout is required")
	} else if _, err := time.ParseDuration(cfg.ManagerService.UserManagement.CleanupTimeout); err != nil {
		errMsgs = append(errMsgs, "manager_service.user_management.cleanup_timeout must be a valid duration string (e.g., 30s, 5m)")
	}

	if cfg.ManagerService.AuthService.ConnectionTimeout == "" {
		errMsgs = append(errMsgs, "manager_service.auth_service.connection_timeout is required")
	} else if _, err := time.ParseDuration(cfg.ManagerService.AuthService.ConnectionTimeout); err != nil {
		errMsgs = append(errMsgs, "manager_service.auth_service.connection_timeout must be a valid duration string (e.g., 30s, 5m)")
	}

	if cfg.ManagerService.Security.SessionSecret == "" {
		errMsgs = append(errMsgs, "manager_service.security.session_secret is required")
	}

	if cfg.ManagerService.AdminID == "" {
		errMsgs = append(errMsgs, "manager_service.admin_id is required")
	} else if !ruleset.AllowedDockerID(cfg.ManagerService.AdminID) {
		errMsgs = append(errMsgs, "manager_service.admin_id must be a valid Docker ID")
	}

	if cfg.Volumes.Host.Volumes == "" {
		errMsgs = append(errMsgs, "volumes.host.volumes is required")
	} else if err := UsablePath(cfg.Volumes.Host.Volumes); err != nil {
		errMsgs = append(errMsgs, fmt.Sprintf("volumes.host.volumes (%v)", err))
	}

	if cfg.Volumes.Host.Homes == "" {
		errMsgs = append(errMsgs, "volumes.host.homes is required")
	} else if err := UsablePath(cfg.Volumes.Host.Homes); err != nil {
		errMsgs = append(errMsgs, fmt.Sprintf("volumes.host.homes (%v)", err))
	}

	if cfg.Volumes.Host.Share == "" {
		errMsgs = append(errMsgs, "volumes.host.share is required")
	} else if err := UsablePath(cfg.Volumes.Host.Share); err != nil {
		errMsgs = append(errMsgs, fmt.Sprintf("volumes.host.share (%v)", err))
	}

	if cfg.Volumes.Host.Readonly == "" {
		errMsgs = append(errMsgs, "volumes.host.readonly is required")
	} else if err := UsablePath(cfg.Volumes.Host.Readonly); err != nil {
		errMsgs = append(errMsgs, fmt.Sprintf("volumes.host.readonly (%v)", err))
	}

	if cfg.Volumes.Container.Share == "" {
		errMsgs = append(errMsgs, "volumes.container.share_dir is required")
	} else if !strings.HasPrefix(cfg.Volumes.Container.Share, "/") {
		errMsgs = append(errMsgs, "volumes.container.share_dir must start with '/'")
	}

	if cfg.Volumes.Container.Readonly == "" {
		errMsgs = append(errMsgs, "volumes.container.readonly_dir is required")
	} else if !strings.HasPrefix(cfg.Volumes.Container.Readonly, "/") {
		errMsgs = append(errMsgs, "volumes.container.readonly_dir must start with '/'")
	}

	if cfg.Volumes.DiskLimit == "" {
		errMsgs = append(errMsgs, "volumes.disk_limit is required")
	} else if _, err := convert.BytesFromString(cfg.Volumes.DiskLimit); err != nil {
		errMsgs = append(errMsgs, "volumes.disk_limit must be a valid size string (e.g., 1g, 512m)")
	}

	if len(errMsgs) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(errMsgs, "; "))
	}

	return nil
}

// UsablePath checks if the given path is usable (exists or can be created).
func UsablePath(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(path, 0755); err != nil {
				return fmt.Errorf("path does not exist and cannot be created: %s", path)
			} else {
				// Clean up the created directory if it was just for validation
				defer os.RemoveAll(path)
			}
			return nil
		}
		return err
	}
	return nil
}

// validateLimits checks if at least one limit value is non-zero.
func validateLimits(l Limits) error {
	errMsgs := []string{}

	nanoCPU, err := convert.NanoCPUsFromString(fmt.Sprintf("%v", l.CPU))
	if err != nil {
		errMsgs = append(errMsgs, "cpu limit must be a valid numeric string (e.g., 1, 0.5)")
	} else if nanoCPU <= 0 {
		errMsgs = append(errMsgs, "cpu limit must be greater than zero")
	}

	mem, err := convert.BytesFromString(l.Memory)
	if err != nil {
		errMsgs = append(errMsgs, "memory limit must be a valid size string (e.g., 512m, 1g)")
	} else if mem <= 0 {
		errMsgs = append(errMsgs, "memory limit must be greater than zero")
	}

	if l.PID <= 0 {
		errMsgs = append(errMsgs, "pid limit must be greater than zero")
	}

	disk, err := convert.BytesFromString(l.Disk)
	if err != nil {
		errMsgs = append(errMsgs, "disk limit must be a valid size string (e.g., 1g, 512m)")
	} else if disk <= 0 {
		errMsgs = append(errMsgs, "disk limit must be greater than zero")
	}

	if l.Ulimits.Nofile.Soft <= 0 {
		errMsgs = append(errMsgs, "ulimits.nofile.soft must be greater than zero")
	}
	if l.Ulimits.Nofile.Hard <= 0 {
		errMsgs = append(errMsgs, "ulimits.nofile.hard must be greater than zero")
	}
	if l.Ulimits.Nofile.Hard < l.Ulimits.Nofile.Soft {
		errMsgs = append(errMsgs, "ulimits.nofile.hard must be greater than or equal to ulimits.nofile.soft")
	}

	if len(errMsgs) > 0 {
		return fmt.Errorf(strings.Join(errMsgs, "; "))
	}

	return nil
}
