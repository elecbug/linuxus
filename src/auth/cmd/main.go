package main

import (
	"authserver/internal/handler"
	"authserver/internal/user"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	authListFile, err := getEnv("AUTH_LIST")
	if err != nil {
		log.Fatalf("failed to get environment variable: %v", err)
	}
	sessionSecret, err := getEnv("SESSION_SECRET")
	if err != nil {
		log.Fatalf("failed to get environment variable: %v", err)
	}
	loginPath, err := getEnv("LOGIN_PATH")
	if err != nil {
		log.Fatalf("failed to get environment variable: %v", err)
	}
	logoutPath, err := getEnv("LOGOUT_PATH")
	if err != nil {
		log.Fatalf("failed to get environment variable: %v", err)
	}
	servicePath, err := getEnv("SERVICE_PATH")
	if err != nil {
		log.Fatalf("failed to get environment variable: %v", err)
	}
	terminalPath, err := getEnv("TERMINAL_PATH")
	if err != nil {
		log.Fatalf("failed to get environment variable: %v", err)
	}
	adminLoginID, err := getEnv("ADMIN_LOGIN_ID")
	if err != nil {
		log.Fatalf("failed to get environment variable: %v", err)
	}
	adminLoginPassword, err := getEnv("ADMIN_LOGIN_PASSWORD")
	if err != nil {
		log.Fatalf("failed to get environment variable: %v", err)
	}
	adminContainerName, err := getEnv("ADMIN_CONTAINER_NAME")
	if err != nil {
		log.Fatalf("failed to get environment variable: %v", err)
	}

	users, err := user.LoadUsers(authListFile)
	if err != nil {
		log.Fatalf("failed to load users: %v", err)
	}

	mux := http.NewServeMux()

	app := handler.NewApp(users, []byte(sessionSecret),
		loginPath, logoutPath, servicePath, terminalPath, adminLoginID, adminLoginPassword, adminContainerName)
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
