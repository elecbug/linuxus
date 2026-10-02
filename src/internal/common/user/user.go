package user

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/elecbug/linuxus/src/internal/common/ruleset"
)

// LoadUsers reads user credentials from the auth list file.
func LoadUsers(path string) (map[string]string, error) {
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

		if !ruleset.AllowedUserID(id) || hash == "" {
			return nil, fmt.Errorf("invalid line in auths file: %s", line)
		}

		users[id] = hash
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

// AddUser adds a new user with a bcrypt-hashed password to memory and file.
func AddUser(path string, users map[string]string, id, password string) error {
	if _, ok := users[id]; ok {
		return fmt.Errorf("user '%s' already exists", id)
	}

	if !ruleset.AllowedUserID(id) || password == "" {
		return fmt.Errorf("invalid user ID or password")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}

	if err := EnsureFile(path); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("failed to open auth file: %v", err)
	}
	defer file.Close()

	prefix := ""
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() > 0 {
		last := make([]byte, 1)
		if _, err := file.ReadAt(last, info.Size()-1); err != nil {
			return err
		}
		if last[0] != '\n' {
			prefix = "\n"
		}
	}
	if _, err := file.WriteString(fmt.Sprintf("%s%s:%s\n", prefix, id, string(hash))); err != nil {
		return fmt.Errorf("failed to write to auth file: %v", err)
	}
	users[id] = string(hash)

	return nil
}

// RemoveUser deletes a user from memory and the auth list file.
func RemoveUser(path string, users map[string]string, id string) error {
	if _, ok := users[id]; !ok {
		return fmt.Errorf("user '%s' does not exist", id)
	}

	file, err := os.OpenFile(path, os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("failed to open auth file: %v", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			lines = append(lines, line)
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			lines = append(lines, line)
			continue
		}

		existingID := strings.TrimSpace(parts[0])
		if existingID != id {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read auth file: %v", err)
	}

	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate auth file: %v", err)
	}

	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek auth file: %v", err)
	}

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("failed to write to auth file: %v", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush auth file: %v", err)
	}
	delete(users, id)

	return nil
}

// SyncUsers reloads the user credentials from the auth list file into memory.
func SyncUsers(users map[string]string, authListPath string) error {
	loadedUsers, err := LoadUsers(authListPath)
	if err != nil {
		return fmt.Errorf("failed to sync users: %v", err)
	}

	clear(users)
	for id, hash := range loadedUsers {
		users[id] = hash
	}

	return nil
}

// ExistsUser checks if a user ID exists in the provided user map.
func ExistsUser(users map[string]string, id string) bool {
	_, ok := users[id]
	return ok
}

// EnsureFile initializes a new deployment without truncating existing credentials.
func EnsureFile(path string) error {
	if path == "" {
		return fmt.Errorf("auth list path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create auth directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("initialize auth list: %w", err)
	}
	return f.Close()
}
