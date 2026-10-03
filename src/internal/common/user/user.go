package user

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/crypto/bcrypt"

	"github.com/elecbug/linuxus/src/internal/common/ruleset"
)

// openLocked locks the auth file itself so host commands and the container's
// bind mount synchronize on the same inode. Closing the file releases the lock.
func openLocked(path string, flags, operation int) (*os.File, error) {
	file, err := os.OpenFile(path, flags, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), operation); err != nil {
		file.Close()
		return nil, fmt.Errorf("lock auth list: %w", err)
	}
	return file, nil
}

// LoadUsers reads a complete credential snapshot, waiting for active writers.
func LoadUsers(path string) (map[string]string, error) {
	file, err := openLocked(path, os.O_RDONLY, syscall.LOCK_SH)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return readUsers(file)
}

func readUsers(reader io.Reader) (map[string]string, error) {
	users := make(map[string]string)
	scanner := bufio.NewScanner(reader)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid auth list entry at line %d", lineNumber)
		}
		id, hash := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if !ruleset.AllowedUserID(id) || hash == "" {
			return nil, fmt.Errorf("invalid auth list entry at line %d", lineNumber)
		}
		if _, exists := users[id]; exists {
			return nil, fmt.Errorf("duplicate auth list entry at line %d", lineNumber)
		}
		users[id] = hash
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

// AddUser rechecks the file under an exclusive lock instead of trusting a stale
// caller snapshot. Callers must synchronize access to their in-memory map.
func AddUser(path string, users map[string]string, id, password string) error {
	if users == nil {
		return fmt.Errorf("user map is not initialized")
	}
	if !ruleset.AllowedUserID(id) || password == "" {
		return fmt.Errorf("invalid user ID or password")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	if err := EnsureFile(path); err != nil {
		return err
	}
	file, err := openLocked(path, os.O_RDWR, syscall.LOCK_EX)
	if err != nil {
		return fmt.Errorf("failed to open auth file: %w", err)
	}
	defer file.Close()
	current, err := readUsers(file)
	if err != nil {
		return err
	}
	if _, exists := current[id]; exists {
		return fmt.Errorf("user %q already exists", id)
	}

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
	entry := []byte(fmt.Sprintf("%s%s:%s\n", prefix, id, string(hash)))
	if err := appendAuthEntry(file, info.Size(), entry); err != nil {
		return err
	}
	current[id] = string(hash)
	replaceUsers(users, current)
	return nil
}

// RemoveUser preserves the inode used by Docker's file bind mount. Readers and
// other writers cannot observe or overwrite the in-place update while locked.
func RemoveUser(path string, users map[string]string, id string) error {
	if users == nil {
		return fmt.Errorf("user map is not initialized")
	}
	if !ruleset.AllowedUserID(id) {
		return fmt.Errorf("invalid user ID")
	}
	file, err := openLocked(path, os.O_RDWR, syscall.LOCK_EX)
	if err != nil {
		return fmt.Errorf("failed to open auth file: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	current, err := readUsers(bytes.NewReader(data))
	if err != nil {
		return err
	}
	if _, exists := current[id]; !exists {
		return fmt.Errorf("user %q does not exist", id)
	}

	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		parts := strings.SplitN(trimmed, ":", 2)
		if !strings.HasPrefix(trimmed, "#") && len(parts) == 2 && strings.TrimSpace(parts[0]) == id {
			continue
		}
		lines = append(lines, line)
	}
	updated := []byte(strings.Join(lines, "\n"))
	if err := replaceAuthData(file, data, updated); err != nil {
		return err
	}
	delete(current, id)
	replaceUsers(users, current)
	return nil
}

func replaceUsers(users, loaded map[string]string) {
	clear(users)
	for id, hash := range loaded {
		users[id] = hash
	}
}

// SyncUsers reloads the user credentials from the auth list file into memory.
func SyncUsers(users map[string]string, authListPath string) error {
	if users == nil {
		return fmt.Errorf("user map is not initialized")
	}
	loaded, err := LoadUsers(authListPath)
	if err != nil {
		return fmt.Errorf("failed to sync users: %w", err)
	}
	replaceUsers(users, loaded)
	return nil
}

// ExistsUser checks if a user ID exists in the provided user map.
func ExistsUser(users map[string]string, id string) bool { _, ok := users[id]; return ok }

// EnsureFile initializes a new deployment without truncating existing credentials.
func EnsureFile(path string) error {
	if path == "" {
		return fmt.Errorf("auth list path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create auth directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("initialize auth list: %w", err)
	}
	return file.Close()
}
