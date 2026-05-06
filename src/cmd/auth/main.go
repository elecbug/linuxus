package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/elecbug/linuxus/src/internal/auth/handler"
	"github.com/elecbug/linuxus/src/internal/common/config"
	"github.com/elecbug/linuxus/src/internal/common/user"
)

// main loads configuration, registers routes, and starts the auth server.
func main() {
	config, err := parseConfig()
	if err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	app := handler.NewApp(config)
	app.RegisterRoutes()

	if err := app.Start(":8080"); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

// parseConfig loads all runtime settings from environment variables and auth list data.
func parseConfig() (*handler.AppConfig, error) {
	var err error

	env := os.Getenv("ENV")
	if env == "" {
		return nil, fmt.Errorf("ENV environment variable is required")
	}

	var cfg config.Config

	err = json.Unmarshal([]byte(env), &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ENV variable as JSON: %v", err)
	}

	users, err := user.LoadUsers(cfg.AuthService.Mounts.ContainerAuthListPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load users: %v", err)
	}

	timeout, err := time.ParseDuration(cfg.ManagerService.AuthService.ConnectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manager connection timeout: %v", err)
	}

	return &handler.AppConfig{
		Users:                   users,
		AuthListFile:            cfg.AuthService.Mounts.ContainerAuthListPath,
		SessionKey:              []byte(cfg.AuthService.Security.SessionSecret),
		LoginPath:               cfg.AuthService.ServiceURL.Login,
		LogoutPath:              cfg.AuthService.ServiceURL.Logout,
		ServicePath:             cfg.AuthService.ServiceURL.Service,
		TerminalPath:            cfg.AuthService.ServiceURL.Terminal,
		SignupPath:              cfg.AuthService.ServiceURL.Signup,
		UserContainerNamePrefix: cfg.UserService.Container.NamePrefix,
		TrustedProxies:          trustProxiesToSlice(cfg.AuthService.Security.TrustedProxies),
		ManagerBaseURL:          fmt.Sprintf("http://%s:5959", cfg.ManagerService.Container.Name),
		ManagerTimeout:          timeout,
		ManagerSessionSecret:    cfg.ManagerService.Security.SessionSecret,
		AllowSignup:             cfg.AuthService.AllowSignup,
	}, nil
}

// trustProxiesToSlice parses a comma-separated trusted proxy list into CIDR strings.
func trustProxiesToSlice(trustedProxies string) []string {
	var trustedProxyCIDRs []string

	if tp := trustedProxies; tp != "" {
		for _, cidr := range strings.Split(tp, ",") {
			cidr = strings.TrimSpace(cidr)
			if cidr != "" {
				trustedProxyCIDRs = append(trustedProxyCIDRs, cidr)
			}
		}
	}

	return trustedProxyCIDRs
}
