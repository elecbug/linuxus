package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/elecbug/linuxus/src/auth/internal/handler"
	"github.com/elecbug/linuxus/src/auth/internal/user"
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

	authListFile, err := getEnv("AUTH_LIST")
	if err != nil {
		return nil, fmt.Errorf("failed to get environment variable: %v", err)
	}
	sessionSecret, err := getEnv("SESSION_SECRET")
	if err != nil {
		return nil, fmt.Errorf("failed to get environment variable: %v", err)
	}
	loginPath, err := getEnv("LOGIN_PATH")
	if err != nil {
		return nil, fmt.Errorf("failed to get environment variable: %v", err)
	}
	logoutPath, err := getEnv("LOGOUT_PATH")
	if err != nil {
		return nil, fmt.Errorf("failed to get environment variable: %v", err)
	}
	servicePath, err := getEnv("SERVICE_PATH")
	if err != nil {
		return nil, fmt.Errorf("failed to get environment variable: %v", err)
	}
	terminalPath, err := getEnv("TERMINAL_PATH")
	if err != nil {
		return nil, fmt.Errorf("failed to get environment variable: %v", err)
	}
	userContainerNamePrefix, err := getEnv("USER_CONTAINER_NAME_PREFIX")
	if err != nil {
		return nil, fmt.Errorf("failed to get environment variable: %v", err)
	}
	trustedProxies := os.Getenv("TRUSTED_PROXIES")
	managerBaseURL, err := getEnv("MANAGER_BASE_URL")
	if err != nil {
		return nil, fmt.Errorf("failed to get environment variable: %v", err)
	}
	managerTimeoutStr, err := getEnv("MANAGER_TIMEOUT")
	if err != nil {
		return nil, fmt.Errorf("failed to get environment variable: %v", err)
	}
	managerTimeout, err := time.ParseDuration(managerTimeoutStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MANAGER_TIMEOUT: %v", err)
	}
	managerSecret, err := getEnv("MANAGER_SECRET")
	if err != nil {
		return nil, fmt.Errorf("failed to get environment variable: %v", err)
	}
	allowSignupStr, err := getEnv("ALLOW_SIGNUP")
	if err != nil {
		return nil, fmt.Errorf("failed to get environment variable: %v", err)
	}
	allowSignup := false
	if allowSignupStr != "" {
		if allowSignupStr == fmt.Sprintf("%v", true) {
			allowSignup = true
		}
	}

	users, err := user.LoadUsers(authListFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load users: %v", err)
	}

	return &handler.AppConfig{
		Users:                   users,
		SessionKey:              []byte(sessionSecret),
		LoginPath:               loginPath,
		LogoutPath:              logoutPath,
		ServicePath:             servicePath,
		TerminalPath:            terminalPath,
		UserContainerNamePrefix: userContainerNamePrefix,
		TrustedProxies:          trustProxiesToSlice(trustedProxies),
		ManagerBaseURL:          managerBaseURL,
		ManagerTimeout:          managerTimeout,
		ManagerSecret:           managerSecret,
		AllowSignup:             allowSignup,
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

// getEnv returns a required environment variable or an error if it is missing.
func getEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("environment variable %s not set", key)
	}
	return value, nil
}
