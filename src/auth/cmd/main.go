package main

import (
	"fmt"
	"log"
	"os"

	"github.com/elecbug/linuxus/src/auth/internal/handler"
	"github.com/elecbug/linuxus/src/auth/internal/user"
)

func main() {
	var authListFile,
		sessionSecret,
		loginPath,
		logoutPath,
		servicePath,
		terminalPath,
		adminUserID,
		userContainerNamePrefix,
		trustedProxies = getEnvs()

	users, err := user.LoadUsers(authListFile)
	if err != nil {
		log.Fatalf("failed to load users: %v", err)
	}

	trustedProxyCIDRs := handler.ParseTrustedProxies(trustedProxies)

	app := handler.NewApp(
		users,
		[]byte(sessionSecret),
		loginPath,
		logoutPath,
		servicePath,
		terminalPath,
		adminUserID,
		userContainerNamePrefix,
		trustedProxyCIDRs,
	)
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

func getEnvs() (
	authListFile,
	sessionSecret,
	loginPath,
	logoutPath,
	servicePath,
	terminalPath,
	adminUserID,
	userContainerNamePrefix,
	trustedProxies string,
) {
	var err error

	authListFile, err = getEnv("AUTH_LIST")
	if err != nil {
		log.Fatalf("failed to get environment variable: %v", err)
	}
	sessionSecret, err = getEnv("SESSION_SECRET")
	if err != nil {
		log.Fatalf("failed to get environment variable: %v", err)
	}
	loginPath, err = getEnv("LOGIN_PATH")
	if err != nil {
		log.Fatalf("failed to get environment variable: %v", err)
	}
	logoutPath, err = getEnv("LOGOUT_PATH")
	if err != nil {
		log.Fatalf("failed to get environment variable: %v", err)
	}
	servicePath, err = getEnv("SERVICE_PATH")
	if err != nil {
		log.Fatalf("failed to get environment variable: %v", err)
	}
	terminalPath, err = getEnv("TERMINAL_PATH")
	if err != nil {
		log.Fatalf("failed to get environment variable: %v", err)
	}
	adminUserID, err = getEnv("ADMIN_USER_ID")
	if err != nil {
		log.Fatalf("failed to get environment variable: %v", err)
	}
	userContainerNamePrefix, err = getEnv("USER_CONTAINER_NAME_PREFIX")
	if err != nil {
		log.Fatalf("failed to get environment variable: %v", err)
	}
	trustedProxies = os.Getenv("TRUSTED_PROXIES")

	return
}
