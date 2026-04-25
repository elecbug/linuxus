package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"
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
		if userID == a.Config.ManagerService.AdminID {
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
