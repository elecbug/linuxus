package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/elecbug/linuxus/src/internal/auth/handler"
	"github.com/elecbug/linuxus/src/internal/common/config"
	"github.com/elecbug/linuxus/src/internal/common/user"
)

// Run loads configuration, registers routes, and starts the auth server.
func Run() error {
	config, err := parseConfig()
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	app := handler.NewApp(config)
	defer app.Stop()
	app.RegisterRoutes()

	return app.Start(":8080")
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

	if err := config.ValidateAuthRoutes(&cfg); err != nil {
		return nil, err
	}

	users, err := user.LoadUsers(cfg.AuthService.Mounts.ContainerAuthListPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load users: %v", err)
	}

	timeout, err := time.ParseDuration(cfg.ManagerService.AuthService.ConnectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manager connection timeout: %v", err)
	}

	templates, classes, err := config.ParseTemplates(&cfg)
	if err != nil {
		return nil, err
	}
	return &handler.AppConfig{
		AdminID: cfg.ManagerService.AdminID, Templates: templates, Classes: classes,
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
