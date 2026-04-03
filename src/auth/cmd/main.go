package main

import (
	"authserver/internal/handler"
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	authListFile := getEnv("AUTH_LIST", "/data/auths.txt")
	sessionSecret := getEnv("SESSION_SECRET", "replace-this-with-a-long-random-secret-key")
	loginPath := getEnv("LOGIN_PATH", "login")
	logoutPath := getEnv("LOGOUT_PATH", "logout")
	servicePath := getEnv("SERVICE_PATH", "service")
	terminalPath := getEnv("TERMINAL_PATH", "terminal")
	adminLoginID := getEnv("ADMIN_LOGIN_ID", "admin")
	adminLoginPassword := getEnv("ADMIN_LOGIN_PASSWORD", "admin")
	adminContainerName := getEnv("ADMIN_CONTAINER_NAME", "manager")

	users, err := loadUsers(authListFile)
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

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func loadUsers(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	users := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line in auths file: %s", line)
		}

		studentID := strings.TrimSpace(parts[0])
		hash := strings.TrimSpace(parts[1])

		if studentID == "" || hash == "" {
			return nil, fmt.Errorf("invalid line in auths file: %s", line)
		}

		users[studentID] = hash
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
