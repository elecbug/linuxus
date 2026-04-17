package app

import (
	"regexp"
	"strings"
)

// sanitizeName normalizes an identifier into a safe lowercase token.
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

// strPtr returns a pointer to the provided string value.
func strPtr(s string) *string {
	return &s
}
