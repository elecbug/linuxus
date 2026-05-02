package app

import "strings"

// isAllUsersKeyword checks if the provided string matches any of the defined keywords for "all users".
func isAllUsersKeyword(s string) bool {
	allUserKeywords := []string{"--all", "-a"}

	s = strings.ToLower(s)
	for _, keyword := range allUserKeywords {
		if s == strings.ToLower(keyword) {
			return true
		}
	}
	return false
}

// isKeyword checks if the provided string is a CLI option (starts with "-" or "--").
func isKeyword(s string) bool {
	s = strings.ToLower(s)
	if strings.HasPrefix(s, "--") || strings.HasPrefix(s, "-") {
		return true
	}
	return false
}
