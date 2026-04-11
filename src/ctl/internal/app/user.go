package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func (a *App) LoadUsers() error {
	authList := a.Config.AuthService.ListFile

	f, err := os.Open(authList)
	if err != nil {
		return fmt.Errorf("AUTH_LIST_FILE not found: %s", authList)
	}
	defer f.Close()

	a.UserIDs = nil
	a.SafeIDs = nil
	a.Seen = make(map[string]struct{})

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
		if userID == a.Config.Admin.UserID {
			continue
		}
		if _, exists := a.Seen[userID]; exists {
			fmt.Fprintf(os.Stderr, "Warning: duplicate user ID skipped: %s\n", userID)
			continue
		}

		safeID := sanitizeName(userID)
		a.Seen[userID] = struct{}{}
		a.UserIDs = append(a.UserIDs, userID)
		a.SafeIDs = append(a.SafeIDs, safeID)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed reading auth list: %w", err)
	}
	if len(a.UserIDs) == 0 {
		return fmt.Errorf("no valid user IDs found in %s", authList)
	}
	return nil
}
