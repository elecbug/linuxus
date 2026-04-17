package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/elecbug/linuxus/src/auth/internal/handler"
	"github.com/elecbug/linuxus/src/auth/internal/user"
)

// main boots the auth service with environment-derived configuration.
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

// getEnv returns a required environment variable or an error when empty.
func getEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("environment variable %s not set", key)
	}
	return value, nil
}

// parseConfig reads environment values and constructs the auth application config.
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
		TrustedProxies:          handler.ParseTrustedProxies(trustedProxies),
		ManagerBaseURL:          managerBaseURL,
		ManagerTimeout:          managerTimeout,
	}, nil
}
