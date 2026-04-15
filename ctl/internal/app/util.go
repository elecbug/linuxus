package app

import (
	"regexp"
	"strings"
)

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

func strPtr(s string) *string {
	return &s
}
