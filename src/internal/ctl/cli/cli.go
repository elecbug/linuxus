package cli

import (
	"fmt"
	"strings"
)

const TRUE_STR = "true"
const FALSE_STR = "false"

// Parameters represents a structured way to handle CLI parameters, including both key-value pairs and a main parameter for commands that require a primary argument.
type Parameters struct {
	// Params stores key-value pairs of CLI options and their associated values.
	Params map[string]string
	// MainParam captures the primary argument for commands that require a main parameter (e.g., a username for add-user).
	MainParam string
}

// NewParameters creates a new Parameters instance with an initialized map.
func NewParameters() *Parameters {
	return &Parameters{
		Params: make(map[string]string),
	}
}

// IsKeyword checks if the provided string is a CLI option (starts with "-" or "--").
func IsKeyword(s string) bool {
	s = strings.ToLower(s)
	if (strings.HasPrefix(s, "--") || strings.HasPrefix(s, "-")) && !strings.HasPrefix(s, "---") {
		return true
	}
	return false
}

// ParseParams converts CLI arguments into a structured Parameters instance, separating key-value pairs and the main parameter.
func ParseParams(params []string) (*Parameters, error) {
	result := NewParameters()
	for i := 0; i < len(params); i++ {
		param := params[i]
		if !strings.HasPrefix(param, "-") {
			if result.MainParam != "" || param == "" {
				return nil, fmt.Errorf("unexpected positional argument %q", param)
			}
			result.MainParam = param
			continue
		}
		key, value, hasValue := strings.Cut(param, "=")
		switch key {
		case "--all", "-a":
			key = "all"
			if hasValue {
				return nil, fmt.Errorf("--all does not accept a value")
			}
			value = TRUE_STR
		case "--user", "-u":
			key = "user"
			if !hasValue {
				if i+1 == len(params) || strings.HasPrefix(params[i+1], "-") {
					return nil, fmt.Errorf("--user requires a username")
				}
				i++
				value = params[i]
			}
			if value == "" {
				return nil, fmt.Errorf("--user requires a username")
			}
		default:
			return nil, fmt.Errorf("unknown option %q", key)
		}
		if _, exists := result.Params[key]; exists {
			return nil, fmt.Errorf("duplicate option --%s", key)
		}
		result.Params[key] = value
	}
	return result, nil
}
