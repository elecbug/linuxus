package main

import (
	"authserver/internal/handler"
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
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

func getEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("environment variable %s not set", key)
	}
	return value, nil
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

		id := strings.TrimSpace(parts[0])
		hash := strings.TrimSpace(parts[1])

		if id == "" || hash == "" {
			return nil, fmt.Errorf("invalid line in auths file: %s", line)
		}

		users[id] = hash
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func addUser(path string, users map[string]string, id, password string) error {
	if _, ok := users[id]; ok {
		return fmt.Errorf("user '%s' already exists", id)
	}

	if id == "" || password == "" {
		return fmt.Errorf("invalid user ID or password")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}

	users[id] = string(hash)

	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open auth file: %v", err)
	}
	defer file.Close()

	if _, err := file.WriteString(fmt.Sprintf("%s:%s\n", id, string(hash))); err != nil {
		return fmt.Errorf("failed to write to auth file: %v", err)
	}

	return nil
}
