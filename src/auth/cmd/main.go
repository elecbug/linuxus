package main

import (
	"fmt"
	"log"
	"os"

	"github.com/elecbug/linuxus/src/auth/internal/handler"
	"github.com/elecbug/linuxus/src/auth/internal/user"
)

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

func getEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("environment variable %s not set", key)
	}
	return value, nil
}

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
	}, nil
}
