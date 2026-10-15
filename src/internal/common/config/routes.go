package config

import (
	"fmt"
	"path"
	"strings"
)

// AdminRoute preserves the default route for deployments created before this setting.
func (cfg *Config) AdminRoute() string {
	if cfg.AuthService.ServiceURL.Admin == "" {
		return "admin"
	}
	return cfg.AuthService.ServiceURL.Admin
}

// ValidateAuthRoutes rejects patterns that ServeMux would reinterpret, conflict
// with, or redirect away from. Nested paths such as auth/login remain supported.
func ValidateAuthRoutes(cfg *Config) error {
	routes := []struct{ name, value string }{
		{"admin", cfg.AdminRoute()},
		{"login", cfg.AuthService.ServiceURL.Login},
		{"logout", cfg.AuthService.ServiceURL.Logout},
		{"service", cfg.AuthService.ServiceURL.Service},
		{"terminal", cfg.AuthService.ServiceURL.Terminal},
		{"signup", cfg.AuthService.ServiceURL.Signup},
	}
	seen := make(map[string]string)
	for _, route := range routes {
		value := route.value
		valid := value != "" && !strings.HasPrefix(value, "/") && path.Clean("/"+value) == "/"+value
		for _, char := range value {
			if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("-_.~/", char)) {
				valid = false
				break
			}
		}
		if !valid {
			return fmt.Errorf("AUTH_SERVICE_SERVICE_URL_%s must be a clean URL path without leading/trailing slashes, query, escapes, or wildcard patterns", strings.ToUpper(route.name))
		}
		if value == "healthz" || value == "static" || strings.HasPrefix(value, "static/") || value == "favicon.ico" {
			return fmt.Errorf("AUTH_SERVICE_SERVICE_URL_%s conflicts with a reserved asset route", strings.ToUpper(route.name))
		}
		if previous, exists := seen[value]; exists {
			return fmt.Errorf("AUTH_SERVICE_SERVICE_URL_%s duplicates %s", strings.ToUpper(route.name), strings.ToUpper(previous))
		}
		for previousPath, previousName := range seen {
			if (route.name == "admin" || previousName == "admin") &&
				(strings.HasPrefix(value, previousPath+"/") || strings.HasPrefix(previousPath, value+"/")) {
				return fmt.Errorf("AUTH_SERVICE_SERVICE_URL_%s overlaps the administrator route", strings.ToUpper(route.name))
			}
		}
		seen[value] = route.name
	}
	return nil
}
