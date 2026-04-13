package app

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
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

func getIP(baseIP string, index int) (string, error) {
	parts := strings.Split(baseIP, ".")
	if len(parts) != 4 {
		return "", fmt.Errorf("invalid base IP format: %s", baseIP)
	}

	octets := make([]int, 4)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > 255 {
			return "", fmt.Errorf("invalid base IP format: %s", baseIP)
		}
		octets[i] = n
	}

	if index < 0 {
		return "", errors.New("index must be a non-negative integer")
	}

	thirdOffset := index / 16
	fourthOffset := (index % 16) * 16

	newO3 := octets[2] + thirdOffset
	newO4 := fourthOffset

	if newO3 > 255 {
		return "", errors.New("subnet overflow (3rd octet > 255)")
	}

	return fmt.Sprintf("%d.%d.%d.%d/28", octets[0], octets[1], newO3, newO4), nil
}

func strPtr(s string) *string {
	return &s
}
