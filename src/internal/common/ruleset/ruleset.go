package ruleset

// AllowedUserID checks if the provided ID is valid according to defined rules.
func AllowedUserID(id string) bool {
	if id == "" {
		return false
	}
	for i, ch := range id {
		if i == 0 && (ch == '_' || ch == '-') {
			return false
		}
		if i == len(id)-1 && (ch == '_' || ch == '-') {
			return false
		}
		if (ch >= 'A' && ch <= 'Z') ||
			(ch >= 'a' && ch <= 'z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '_' || ch == '-' {
			continue
		}
		return false
	}
	return true
}

// AllowedDockerID checks if the given string is a valid Docker ID.
func AllowedDockerID(id string) bool {
	if id == "" {
		return false
	}
	for i, ch := range id {
		if i == 0 && (ch == '_' || ch == '-') {
			return false
		}
		if i == len(id)-1 && (ch == '_' || ch == '-') {
			return false
		}
		if (ch >= 'A' && ch <= 'Z') ||
			(ch >= 'a' && ch <= 'z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '_' || ch == '-' {
			continue
		}
		return false
	}
	return true
}

// AllowedDockerPrefix checks if the given string is a valid Docker prefix.
func AllowedDockerPrefix(prefix string) bool {
	if prefix == "" {
		return false
	}
	for i, ch := range prefix {
		if i == 0 && (ch == '_' || ch == '-') {
			return false
		}
		if (ch >= 'A' && ch <= 'Z') ||
			(ch >= 'a' && ch <= 'z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '_' || ch == '-' {
			continue
		}
		return false
	}
	return true
}
