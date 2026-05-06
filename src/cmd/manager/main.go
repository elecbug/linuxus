package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	ctl_config "github.com/elecbug/linuxus/src/internal/common/config"
	"github.com/elecbug/linuxus/src/internal/common/convert"
	"github.com/elecbug/linuxus/src/internal/manager/config"
	"github.com/elecbug/linuxus/src/internal/manager/handler"
)

// main loads environment configuration and starts the manager server.
func main() {
	cfg, err := parseConfigFromEnv()
	if err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	s, err := handler.NewServer(cfg)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	s.RegisterRoutes()
	s.Start()
}

// parseConfigFromEnv reads required and optional manager settings from environment variables.
func parseConfigFromEnv() (*config.Config, error) {
	var err error

	env := os.Getenv("ENV")
	if env == "" {
		return nil, fmt.Errorf("ENV environment variable is required")
	}

	var cfg ctl_config.Config

	err = json.Unmarshal([]byte(env), &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ENV variable as JSON: %v", err)
	}

	userImage := os.Getenv("USER_IMAGE")
	if userImage == "" {
		return nil, fmt.Errorf("USER_IMAGE environment variable is required")
	}

	managerWaitTime, err := time.ParseDuration(cfg.ManagerService.AuthService.ConnectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid ManagerService.AuthService.ConnectionTimeout: %v", err)
	}

	containerTimeout, err := time.ParseDuration(cfg.ManagerService.UserManagement.CleanupTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid ManagerService.UserManagement.CleanupTimeout: %v", err)
	}

	userCPUStr := fmt.Sprintf("%v", cfg.UserService.Limits.User.CPU)
	userCPU, err := convert.NanoCPUsFromString(userCPUStr)
	if err != nil {
		return nil, fmt.Errorf("invalid UserService.Limits.User.CPU: %v", err)
	}

	adminCPUStr := fmt.Sprintf("%v", cfg.UserService.Limits.Admin.CPU)
	adminCPU, err := convert.NanoCPUsFromString(adminCPUStr)
	if err != nil {
		return nil, fmt.Errorf("invalid UserService.Limits.Admin.CPU: %v", err)
	}

	userMemBytes, err := convert.BytesFromString(cfg.UserService.Limits.User.Memory)
	if err != nil {
		return nil, fmt.Errorf("invalid UserService.Limits.User.Memory: %v", err)
	}

	adminMemBytes, err := convert.BytesFromString(cfg.UserService.Limits.Admin.Memory)
	if err != nil {
		return nil, fmt.Errorf("invalid UserService.Limits.Admin.Memory: %v", err)
	}

	return &config.Config{
		ListenAddr:              ":5959",
		UserImage:               userImage,
		UserContainerNamePrefix: cfg.UserService.Container.NamePrefix,
		NetworkPrefix:           cfg.UserService.Container.NetworkNamePrefix,
		BaseIP:                  cfg.UserService.Container.BaseSubnet16,
		AuthContainerName:       cfg.AuthService.Container.Name,
		AdminUserID:             cfg.ManagerService.AdminID,
		ManagerSessionSecret:    cfg.ManagerService.Security.SessionSecret,

		RuntimeUser:          fmt.Sprintf("%d:%d", cfg.UserService.Runtime.UID, cfg.UserService.Runtime.GID),
		ContainerRuntimeUser: cfg.UserService.Runtime.LinuxUsername,
		ContainerHostname:    cfg.UserService.Runtime.LinuxHostname,
		WorkingDir:           "/home/" + cfg.UserService.Runtime.LinuxUsername,
		Timezone:             cfg.UserService.Runtime.Timezone,
		ReadOnlyRootFS:       true,
		ManagerWaitTime:      managerWaitTime,
		ContainerTimeout:     containerTimeout,

		HostHomesDir:         cfg.Volumes.Host.Homes,
		HostShareDir:         cfg.Volumes.Host.Share,
		HostReadonlyDir:      cfg.Volumes.Host.Readonly,
		ContainerShareDir:    cfg.Volumes.Container.Share,
		ContainerReadonlyDir: cfg.Volumes.Container.Readonly,

		UserLimits: config.ResourceLimits{
			NanoCPUs:    userCPU,
			MemoryBytes: userMemBytes,
			PidsLimit:   int64(cfg.UserService.Limits.User.PID),
			NofileSoft:  int64(cfg.UserService.Limits.User.Ulimits.Nofile.Soft),
			NofileHard:  int64(cfg.UserService.Limits.User.Ulimits.Nofile.Hard),
		},
		AdminLimits: config.ResourceLimits{
			NanoCPUs:    adminCPU,
			MemoryBytes: adminMemBytes,
			PidsLimit:   int64(cfg.UserService.Limits.Admin.PID),
			NofileSoft:  int64(cfg.UserService.Limits.Admin.Ulimits.Nofile.Soft),
			NofileHard:  int64(cfg.UserService.Limits.Admin.Ulimits.Nofile.Hard),
		},

		ShareDiskLimit: cfg.Volumes.DiskLimit,
		UserDiskLimit:  cfg.UserService.Limits.User.Disk,
		AdminDiskLimit: cfg.UserService.Limits.Admin.Disk,
	}, nil
}

// envInt64 parses an int64 from an environment variable, returning zero if unset and an error if invalid.
func envInt64(key string) (int64, error) {
	s := os.Getenv(key)
	if s == "" {
		return 0, nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return v, nil
}
