package main

import (
	"fmt"
	"log"
	"net/http"
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
		adminLoginID,
		adminLoginPassword,
		adminContainerName = getEnvs()

	users, err := user.LoadUsers(authListFile)
	if err != nil {
		log.Fatalf("failed to load users: %v", err)
	}

	mux := http.NewServeMux()

	app := handler.NewApp(
		users,
		[]byte(sessionSecret),
		loginPath,
		logoutPath,
		servicePath,
		terminalPath,
		adminLoginID,
		adminLoginPassword,
		adminContainerName,
	)
	app.RegisterRoutes(mux)

	addr := ":8080"
	log.Printf("Auth server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
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
	adminLoginID,
	adminLoginPassword,
	adminContainerName string,
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
	adminLoginID, err = getEnv("ADMIN_LOGIN_ID")
	if err != nil {
		log.Fatalf("failed to get environment variable: %v", err)
	}
	adminLoginPassword, err = getEnv("ADMIN_LOGIN_PASSWORD")
	if err != nil {
		log.Fatalf("failed to get environment variable: %v", err)
	}
	adminContainerName, err = getEnv("ADMIN_CONTAINER_NAME")
	if err != nil {
		log.Fatalf("failed to get environment variable: %v", err)
	}

	return
}
