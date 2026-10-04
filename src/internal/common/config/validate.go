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
		errMsgs = append(errMsgs, "USER_SERVICE_CONTAINER_NAME_PREFIX is required")
	} else if !ruleset.AllowedDockerPrefix(cfg.UserService.Container.NamePrefix) {
		errMsgs = append(errMsgs, "USER_SERVICE_CONTAINER_NAME_PREFIX must be a valid Docker prefix")
	}

	if cfg.UserService.Container.NetworkNamePrefix == "" {
		errMsgs = append(errMsgs, "USER_SERVICE_CONTAINER_NETWORK_NAME_PREFIX is required")
	} else if !ruleset.AllowedDockerPrefix(cfg.UserService.Container.NetworkNamePrefix) {
		errMsgs = append(errMsgs, "USER_SERVICE_CONTAINER_NETWORK_NAME_PREFIX must be a valid Docker prefix")
	}

	if cfg.UserService.Container.BaseSubnet16 == "" {
		errMsgs = append(errMsgs, "USER_SERVICE_CONTAINER_BASE_SUBNET_16 is required")
	} else if !subnet.IsValidSubnet16(cfg.UserService.Container.BaseSubnet16) {
		errMsgs = append(errMsgs, "USER_SERVICE_CONTAINER_BASE_SUBNET_16 must be a valid /16 subnet (x.x.0.0)")
	}

	if cfg.UserService.Runtime.UID <= 0 {
		errMsgs = append(errMsgs, "USER_SERVICE_RUNTIME_UID must be greater than zero")
	}

	if cfg.UserService.Runtime.GID <= 0 {
		errMsgs = append(errMsgs, "USER_SERVICE_RUNTIME_GID must be greater than zero")
	}

	if cfg.UserService.Runtime.LinuxUsername == "" {
		errMsgs = append(errMsgs, "USER_SERVICE_RUNTIME_LINUX_USERNAME is required")
	} else if cfg.UserService.Runtime.LinuxUsername == "root" {
		errMsgs = append(errMsgs, "USER_SERVICE_RUNTIME_LINUX_USERNAME cannot be 'root'")
	} else if !ruleset.AllowedDockerID(cfg.UserService.Runtime.LinuxUsername) {
		errMsgs = append(errMsgs, "USER_SERVICE_RUNTIME_LINUX_USERNAME must be a valid Docker ID")
	}

	if cfg.UserService.Runtime.LinuxHostname == "" {
		errMsgs = append(errMsgs, "USER_SERVICE_RUNTIME_LINUX_HOSTNAME is required")
	} else if !ruleset.AllowedDockerID(cfg.UserService.Runtime.LinuxHostname) {
		errMsgs = append(errMsgs, "USER_SERVICE_RUNTIME_LINUX_HOSTNAME must be a valid Docker ID")
	}

	if cfg.UserService.Runtime.Timezone == "" {
		errMsgs = append(errMsgs, "USER_SERVICE_RUNTIME_TIMEZONE is required")
	}

	if err := validateLimits(cfg.UserService.Limits.User); err != nil {
		errMsgs = append(errMsgs, fmt.Sprintf("USER_SERVICE_LIMITS_USER (%v)", err))
	}

	if err := validateLimits(cfg.UserService.Limits.Admin); err != nil {
		errMsgs = append(errMsgs, fmt.Sprintf("USER_SERVICE_LIMITS_ADMIN (%v)", err))
	}

	if cfg.AuthService.Container.Name == "" {
		errMsgs = append(errMsgs, "AUTH_SERVICE_CONTAINER_NAME is required")
	} else if !ruleset.AllowedDockerID(cfg.AuthService.Container.Name) {
		errMsgs = append(errMsgs, "AUTH_SERVICE_CONTAINER_NAME must be a valid Docker ID")
	}

	if cfg.AuthService.Container.ExternalPort <= 0 || cfg.AuthService.Container.ExternalPort > 65535 {
		errMsgs = append(errMsgs, "AUTH_SERVICE_CONTAINER_EXTERNAL_PORT must be a valid port number (1-65535)")
	}

	if cfg.AuthService.Runtime.Timezone == "" {
		errMsgs = append(errMsgs, "AUTH_SERVICE_RUNTIME_TIMEZONE is required")
	}

	if err := ValidateAuthRoutes(cfg); err != nil {
		errMsgs = append(errMsgs, err.Error())
	}

	if cfg.AuthService.Mounts.HostAuthListPath == "" {
		errMsgs = append(errMsgs, "AUTH_SERVICE_MOUNTS_HOST_AUTH_LIST_PATH is required")
	} else if err := UsablePath(cfg.AuthService.Mounts.HostAuthListPath); err != nil {
		errMsgs = append(errMsgs, fmt.Sprintf("AUTH_SERVICE_MOUNTS_HOST_AUTH_LIST_PATH (%v)", err))
	}

	if cfg.AuthService.Mounts.ContainerAuthListPath == "" {
		errMsgs = append(errMsgs, "AUTH_SERVICE_MOUNTS_CONTAINER_AUTH_LIST_PATH is required")
	} else if !strings.HasPrefix(cfg.AuthService.Mounts.ContainerAuthListPath, "/") {
		errMsgs = append(errMsgs, "AUTH_SERVICE_MOUNTS_CONTAINER_AUTH_LIST_PATH must start with '/'")
	}

	if cfg.AuthService.Security.SessionSecret == "" {
		errMsgs = append(errMsgs, "AUTH_SERVICE_SECURITY_SESSION_SECRET is required")
	}

	if err := subnet.IsValidSubnetList(cfg.AuthService.Security.TrustedProxies); err != nil {
		errMsgs = append(errMsgs, fmt.Sprintf("AUTH_SERVICE_SECURITY_TRUSTED_PROXIES (%v)", err))
	}

	if cfg.ManagerService.Container.Name == "" {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_CONTAINER_NAME is required")
	} else if !ruleset.AllowedDockerID(cfg.ManagerService.Container.Name) {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_CONTAINER_NAME must be a valid Docker ID")
	} else if cfg.ManagerService.Container.Name == cfg.AuthService.Container.Name {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_CONTAINER_NAME cannot be the same as AUTH_SERVICE_CONTAINER_NAME")
	}

	if cfg.ManagerService.Container.Network == "" {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_CONTAINER_NETWORK is required")
	} else if !ruleset.AllowedDockerID(cfg.ManagerService.Container.Network) {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_CONTAINER_NETWORK must be a valid Docker ID")
	} else if cfg.ManagerService.Container.Network == cfg.UserService.Container.NetworkNamePrefix {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_CONTAINER_NETWORK cannot be the same as USER_SERVICE_CONTAINER_NETWORK_NAME_PREFIX")
	}

	if cfg.ManagerService.Container.Subnet == "" {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_CONTAINER_SUBNET is required")
	} else if !subnet.IsValidSubnet(cfg.ManagerService.Container.Subnet) {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_CONTAINER_SUBNET must be a valid subnet")
	}

	if cfg.ManagerService.Container.HomesDir == "" {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_CONTAINER_HOMES_DIR is required")
	} else if !strings.HasPrefix(cfg.ManagerService.Container.HomesDir, "/") {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_CONTAINER_HOMES_DIR must start with '/'")
	}

	if cfg.ManagerService.Container.ShareDir == "" {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_CONTAINER_SHARE_DIR is required")
	} else if !strings.HasPrefix(cfg.ManagerService.Container.ShareDir, "/") {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_CONTAINER_SHARE_DIR must start with '/'")
	}

	if cfg.ManagerService.Container.ReadonlyDir == "" {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_CONTAINER_READONLY_DIR is required")
	} else if !strings.HasPrefix(cfg.ManagerService.Container.ReadonlyDir, "/") {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_CONTAINER_READONLY_DIR must start with '/'")
	}

	if cfg.ManagerService.Container.HomesDir != "" && cfg.ManagerService.Container.ShareDir != "" && cfg.ManagerService.Container.HomesDir == cfg.ManagerService.Container.ShareDir {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_CONTAINER_HOMES_DIR and MANAGER_SERVICE_CONTAINER_SHARE_DIR cannot be the same")
	}

	if cfg.ManagerService.Container.HomesDir != "" && cfg.ManagerService.Container.ReadonlyDir != "" && cfg.ManagerService.Container.HomesDir == cfg.ManagerService.Container.ReadonlyDir {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_CONTAINER_HOMES_DIR and MANAGER_SERVICE_CONTAINER_READONLY_DIR cannot be the same")
	}

	if cfg.ManagerService.Container.ShareDir != "" && cfg.ManagerService.Container.ReadonlyDir != "" && cfg.ManagerService.Container.ShareDir == cfg.ManagerService.Container.ReadonlyDir {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_CONTAINER_SHARE_DIR and MANAGER_SERVICE_CONTAINER_READONLY_DIR cannot be the same")
	}

	if cfg.ManagerService.UserManagement.CleanupTimeout == "" {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_USER_MANAGEMENT_CLEANUP_TIMEOUT is required")
	} else if duration, err := time.ParseDuration(cfg.ManagerService.UserManagement.CleanupTimeout); err != nil || duration < 0 {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_USER_MANAGEMENT_CLEANUP_TIMEOUT must be a non-negative duration (0 disables cleanup)")
	}

	if cfg.ManagerService.AuthService.ConnectionTimeout == "" {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_AUTH_SERVICE_CONNECTION_TIMEOUT is required")
	} else if duration, err := time.ParseDuration(cfg.ManagerService.AuthService.ConnectionTimeout); err != nil || duration <= 0 {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_AUTH_SERVICE_CONNECTION_TIMEOUT must be a positive duration (e.g., 30s, 5m)")
	}

	if cfg.ManagerService.Security.SessionSecret == "" {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_SECURITY_SESSION_SECRET is required")
	}

	if cfg.ManagerService.AdminID == "" {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_ADMIN_ID is required")
	} else if !ruleset.AllowedDockerID(cfg.ManagerService.AdminID) {
		errMsgs = append(errMsgs, "MANAGER_SERVICE_ADMIN_ID must be a valid Docker ID")
	}

	if cfg.Volumes.Host.Volumes == "" {
		errMsgs = append(errMsgs, "VOLUMES_HOST_VOLUMES is required")
	} else if err := UsablePath(cfg.Volumes.Host.Volumes); err != nil {
		errMsgs = append(errMsgs, fmt.Sprintf("VOLUMES_HOST_VOLUMES (%v)", err))
	}

	if cfg.Volumes.Host.Homes == "" {
		errMsgs = append(errMsgs, "VOLUMES_HOST_HOMES is required")
	} else if err := UsablePath(cfg.Volumes.Host.Homes); err != nil {
		errMsgs = append(errMsgs, fmt.Sprintf("VOLUMES_HOST_HOMES (%v)", err))
	}

	if cfg.Volumes.Host.Share == "" {
		errMsgs = append(errMsgs, "VOLUMES_HOST_SHARE is required")
	} else if err := UsablePath(cfg.Volumes.Host.Share); err != nil {
		errMsgs = append(errMsgs, fmt.Sprintf("VOLUMES_HOST_SHARE (%v)", err))
	}

	if cfg.Volumes.Host.Readonly == "" {
		errMsgs = append(errMsgs, "VOLUMES_HOST_READONLY is required")
	} else if err := UsablePath(cfg.Volumes.Host.Readonly); err != nil {
		errMsgs = append(errMsgs, fmt.Sprintf("VOLUMES_HOST_READONLY (%v)", err))
	}

	if cfg.Volumes.Container.Share == "" {
		errMsgs = append(errMsgs, "VOLUMES_CONTAINER_SHARE is required")
	} else if !strings.HasPrefix(cfg.Volumes.Container.Share, "/") {
		errMsgs = append(errMsgs, "VOLUMES_CONTAINER_SHARE must start with '/'")
	}

	if cfg.Volumes.Container.Readonly == "" {
		errMsgs = append(errMsgs, "VOLUMES_CONTAINER_READONLY is required")
	} else if !strings.HasPrefix(cfg.Volumes.Container.Readonly, "/") {
		errMsgs = append(errMsgs, "VOLUMES_CONTAINER_READONLY must start with '/'")
	}

	if cfg.Volumes.DiskLimit == "" {
		errMsgs = append(errMsgs, "VOLUMES_DISK_LIMIT is required")
	} else if size, err := convert.BytesFromString(cfg.Volumes.DiskLimit); err != nil || size <= 1024*1024 {
		errMsgs = append(errMsgs, "VOLUMES_DISK_LIMIT must be a valid size greater than 1MiB (e.g., 1g, 512m)")
	}

	if err := validateVolumePaths(cfg); err != nil {
		errMsgs = append(errMsgs, err.Error())
	}

	if len(errMsgs) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(errMsgs, "; "))
	}

	return nil
}

// UsablePath validates a path without creating or removing directories.
func UsablePath(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
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
	} else if disk <= 1024*1024 {
		errMsgs = append(errMsgs, "disk limit must be greater than 1MiB")
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
		return fmt.Errorf("%s", strings.Join(errMsgs, "; "))
	}

	return nil
}
