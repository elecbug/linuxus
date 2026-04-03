package main

import (
	"authserver/internal/handler"
	"bufio"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	studentsFile := getEnv("STUDENTS_FILE", "/data/students.txt")
	sessionSecret := getEnv("SESSION_SECRET", "replace-this-with-a-long-random-secret-key")

	users, err := loadUsers(studentsFile)
	if err != nil {
		log.Fatalf("failed to load users: %v", err)
	}

	loginTmpl, err := template.New(handler.LOGIN_PATH).Parse(handler.LOGIN_PAGE)
	if err != nil {
		log.Fatalf("failed to parse template: %v", err)
	}

	serviceTmpl, err := template.New(handler.SERVICE_PATH).Parse(handler.SERVICE_PAGE)
	if err != nil {
		log.Fatalf("failed to parse service template: %v", err)
	}

	mux := http.NewServeMux()

	app := handler.NewApp(users, []byte(sessionSecret), loginTmpl, serviceTmpl)
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
			return nil, fmt.Errorf("invalid line in students file: %s", line)
		}

		studentID := strings.TrimSpace(parts[0])
		hash := strings.TrimSpace(parts[1])

		if studentID == "" || hash == "" {
			return nil, fmt.Errorf("invalid line in students file: %s", line)
		}

		users[studentID] = hash
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
