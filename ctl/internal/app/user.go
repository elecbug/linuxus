package app

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// LoadUserList parses user IDs from the auth list file.
func (a *App) LoadUserList() error {
	authList := a.Config.AuthService.AuthListFile.HostPath

	f, err := os.Open(authList)
	if err != nil {
		return fmt.Errorf("AUTH_LIST_FILE not found: %s", authList)
	}
	defer f.Close()

	a.UserIDs = nil
	a.SafeIDs = nil
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
		if userID == a.Config.UserService.Container.Admin.UserID {
			continue
		}
		if _, exists := a.seen[userID]; exists {
			fmt.Fprintf(os.Stderr, "Warning: duplicate user ID skipped: %s\n", userID)
			continue
		}

		safeID := sanitizeName(userID)
		a.seen[userID] = struct{}{}
		a.UserIDs = append(a.UserIDs, userID)
		a.SafeIDs = append(a.SafeIDs, safeID)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed reading auth list: %w", err)
	}
	if len(a.UserIDs) == 0 {
		fmt.Println("Warning: no valid user IDs found in auth list")
	}
	return nil
}

// sanitizeName converts an arbitrary user ID to a Docker-safe identifier.
func sanitizeName(s string) string {
	s = strings.ToLower(s)
	reInvalid := regexp.MustCompile(`[^a-z0-9]+`)
	s = reInvalid.ReplaceAllString(s, "_")
	reDup := regexp.MustCompile(`_+`)
	s = reDup.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		return "invalid"
	}
	return s
}
