package config

import (
	"bufio"
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// ParseEnv parses deployment settings without reading or modifying the process
// environment. Missing settings keep their zero values; ValidateConfig checks
// required fields after host paths have been resolved by the caller.
func ParseEnv(data []byte) (Config, error) {
	var cfg Config
	fields := envFields(reflect.ValueOf(&cfg).Elem(), "")
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(bytes.NewReader(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") || strings.HasPrefix(line, "export\t") {
			line = strings.TrimSpace(line[len("export"):])
		}
		key, raw, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || !validEnvKey(key) {
			return Config{}, fmt.Errorf("line %d: expected KEY=VALUE", lineNumber)
		}
		field, exists := fields[key]
		if !exists {
			return Config{}, fmt.Errorf("line %d: unknown setting %s", lineNumber, key)
		}
		if seen[key] {
			return Config{}, fmt.Errorf("line %d: duplicate setting %s", lineNumber, key)
		}
		value, err := parseEnvValue(strings.TrimSpace(raw))
		if err != nil {
			return Config{}, fmt.Errorf("line %d (%s): %w", lineNumber, key, err)
		}
		if err := setEnvField(field, value); err != nil {
			return Config{}, fmt.Errorf("line %d (%s): %w", lineNumber, key, err)
		}
		seen[key] = true
	}
	if err := scanner.Err(); err != nil {
		return Config{}, fmt.Errorf("read .env at line %d: %w", lineNumber+1, err)
	}
	if len(seen) == 0 {
		return Config{}, fmt.Errorf(".env contains no settings")
	}
	return cfg, nil
}

func envFields(value reflect.Value, prefix string) map[string]reflect.Value {
	fields := make(map[string]reflect.Value)
	for i := 0; i < value.NumField(); i++ {
		name := value.Type().Field(i).Tag.Get("env")
		if name == "" {
			continue
		}
		if prefix != "" {
			name = prefix + "_" + name
		}
		field := value.Field(i)
		if field.Kind() == reflect.Struct {
			for key, child := range envFields(field, name) {
				fields[key] = child
			}
		} else {
			fields[name] = field
		}
	}
	return fields
}

func validEnvKey(key string) bool {
	if key == "" {
		return false
	}
	for i, ch := range key {
		if ch == '_' || ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z' || i > 0 && ch >= '0' && ch <= '9' {
			continue
		}
		return false
	}
	return true
}

func setEnvField(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Interface:
		// CPU limits accept numeric text, retaining the existing JSON transport.
		field.Set(reflect.ValueOf(value))
	case reflect.Int:
		number, err := strconv.ParseInt(value, 10, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("expected an integer in range")
		}
		field.SetInt(number)
	case reflect.Bool:
		if value != "true" && value != "false" {
			return fmt.Errorf("expected true or false")
		}
		field.SetBool(value == "true")
	default:
		return fmt.Errorf("unsupported setting type")
	}
	return nil
}

// Values are literal; $VAR and $(commands) are never expanded or executed.
// Quoted values support trailing comments. Unquoted # starts a comment only
// at the beginning of the value or after whitespace, preserving secret#suffix.
func parseEnvValue(raw string) (string, error) {
	if strings.ContainsRune(raw, 0) {
		return "", fmt.Errorf("NUL is not allowed")
	}
	if raw == "" {
		return "", nil
	}
	quote := raw[0]
	if quote != '\'' && quote != '"' {
		for i, ch := range raw {
			if ch == '#' && (i == 0 || raw[i-1] == ' ' || raw[i-1] == '\t') {
				return strings.TrimSpace(raw[:i]), nil
			}
		}
		return raw, nil
	}
	var result strings.Builder
	for i := 1; i < len(raw); i++ {
		ch := raw[i]
		if ch == quote {
			tail := strings.TrimSpace(raw[i+1:])
			if tail != "" && !strings.HasPrefix(tail, "#") {
				return "", fmt.Errorf("unexpected text after quoted value")
			}
			return result.String(), nil
		}
		if quote == '"' && ch == '\\' {
			i++
			if i == len(raw) {
				break
			}
			switch raw[i] {
			case 'n':
				result.WriteByte('\n')
			case 'r':
				result.WriteByte('\r')
			case 't':
				result.WriteByte('\t')
			case '\\', '"', '$':
				result.WriteByte(raw[i])
			default:
				result.WriteByte('\\')
				result.WriteByte(raw[i])
			}
		} else {
			result.WriteByte(ch)
		}
	}
	return "", fmt.Errorf("unterminated quoted value")
}
