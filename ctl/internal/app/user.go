package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// LoadUserList parses user IDs from the auth list file.
func (a *App) LoadUserList() error {
	authList := a.Config.AuthService.Mounts.HostAuthListPath

	f, err := os.Open(authList)
	if err != nil && os.IsNotExist(err) {
		err = a.systemAPI.CreateEmptyFile(authList, 0)
		if err != nil {
			return fmt.Errorf("failed to create auth list file: %w", err)
		}

		f, err = os.Open(authList)
		if err != nil {
			return fmt.Errorf("failed to open auth list file: %w", err)
		}
	}
	defer f.Close()

	a.UserIDs = nil
	a.seen = make(map[string]struct{})

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		userID := strings.TrimSpace(strings.SplitN(line, ":", 2)[0])
		if userID == "" {
			continue
		}
		if _, exists := a.seen[userID]; exists {
			fmt.Fprintf(os.Stderr, "Warning: duplicate user ID skipped: %s\n", userID)
			continue
		}

		a.seen[userID] = struct{}{}
		a.UserIDs = append(a.UserIDs, userID)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed reading auth list: %w", err)
	}
	if len(a.UserIDs) == 0 {
		fmt.Println("Warning: no valid user IDs found in auth list")
	}
	return nil
}

// existsUser checks if a user ID is already in the loaded user list.
func (a *App) existsUser(userID string) bool {
	_, exists := a.seen[userID]
	return exists
}

// updateUser adds a new user ID and hashed password to the auth list file and updates the in-memory user list.
func (a *App) updateUser(userID, password string) error {
	if a.existsUser(userID) {
		return fmt.Errorf("user ID already exists: %s", userID)
	}

	f, err := os.OpenFile(a.Config.AuthService.Mounts.HostAuthListPath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open auth list for writing: %w", err)
	}
	defer f.Close()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	line := fmt.Sprintf("%s:%s\n", userID, hashedPassword)
	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("failed to write to auth list: %w", err)
	}

	a.UserIDs = append(a.UserIDs, userID)
	a.seen[userID] = struct{}{}
	return nil
}

// removeUser removes a user ID from the auth list file and updates the in-memory user list.
func (a *App) removeUser(userID string) error {
	if !a.existsUser(userID) {
		return fmt.Errorf("user ID not found in auth list: %s", userID)
	}

	f, err := os.Open(a.Config.AuthService.Mounts.HostAuthListPath)
	if err != nil {
		return fmt.Errorf("failed to open auth list for reading: %w", err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, userID+":") {
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed reading auth list: %w", err)
	}

	f, err = os.OpenFile(a.Config.AuthService.Mounts.HostAuthListPath, os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open auth list for writing: %w", err)
	}
	defer f.Close()

	for _, line := range lines {
		if _, err := f.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("failed to write to auth list: %w", err)
		}
	}

	delete(a.seen, userID)
	for i, id := range a.UserIDs {
		if id == userID {
			a.UserIDs = append(a.UserIDs[:i], a.UserIDs[i+1:]...)
			break
		}
	}

	return nil
}
