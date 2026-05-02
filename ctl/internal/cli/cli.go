package cli

import "strings"

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
func ParseParams(params []string) *Parameters {
	result := make(map[string]string)
	mainParam := ""

	for i := 0; i < len(params); i++ {
		param := params[i]

		if IsKeyword(param) {
			key := strings.TrimLeft(param, "-")

			if i+1 < len(params) && !IsKeyword(params[i+1]) {
				result[key] = params[i+1]
				i++
			} else {
				result[key] = TRUE_STR
			}

			continue
		}

		if mainParam == "" {
			mainParam = param
		}
	}

	return &Parameters{
		Params:    result,
		MainParam: mainParam,
	}
}
