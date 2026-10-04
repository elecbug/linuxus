package config

import (
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/elecbug/linuxus/src/internal/common/convert"
	"github.com/elecbug/linuxus/src/internal/common/ruleset"
)

type Template struct {
	Image string `json:"image"`
	Seed  string `json:"seed,omitempty"`
}

func ParseTemplates(cfg *Config) (map[string]Template, map[string]string, error) {
	templates, classes := map[string]Template{
		"linux":  {Image: "default", Seed: "/opt/linuxus/templates/linux"},
		"c":      {Image: "default", Seed: "/opt/linuxus/templates/c"},
		"python": {Image: "default", Seed: "/opt/linuxus/templates/python"},
	}, map[string]string{}
	decode := func(value string, target any) error {
		if strings.TrimSpace(value) == "" {
			return nil
		}
		decoder := json.NewDecoder(strings.NewReader(value))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(target); err != nil {
			return err
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			return fmt.Errorf("expected one JSON object")
		}
		return nil
	}
	if err := decode(cfg.UserService.Templates, &templates); err != nil {
		return nil, nil, fmt.Errorf("USER_SERVICE_TEMPLATES: %w", err)
	}
	if err := decode(cfg.UserService.Classes, &classes); err != nil {
		return nil, nil, fmt.Errorf("USER_SERVICE_CLASSES: %w", err)
	}
	if templates == nil || classes == nil {
		return nil, nil, fmt.Errorf("templates and classes must be JSON objects")
	}
	for name, template := range templates {
		if name == "default" || !ruleset.AllowedUserID(name) || strings.TrimSpace(template.Image) == "" || strings.ContainsAny(template.Image, " \t\r\n\x00") {
			return nil, nil, fmt.Errorf("invalid template %q", name)
		}
		if template.Seed != "" && (!path.IsAbs(template.Seed) || path.Clean(template.Seed) != template.Seed || template.Seed == "/" || strings.ContainsAny(template.Seed, "\r\n\x00")) {
			return nil, nil, fmt.Errorf("invalid template seed path for %s", name)
		}
	}
	for class, template := range classes {
		if !ruleset.AllowedUserID(class) {
			return nil, nil, fmt.Errorf("invalid class name")
		}
		if _, ok := templates[template]; !ok && template != "default" {
			return nil, nil, fmt.Errorf("class %s selects unknown template %s", class, template)
		}
	}
	return templates, classes, nil
}

func validateOperations(cfg *Config) error {
	if cfg.Capacity.MaxRunning < 0 || cfg.Capacity.MaxPending < 0 || cfg.Capacity.MaxPending > 1<<30 {
		return fmt.Errorf("capacity limits must be non-negative")
	}
	if cfg.Capacity.MinFreeSpace != "" {
		if size, err := convert.BytesFromString(cfg.Capacity.MinFreeSpace); err != nil || size < 0 {
			return fmt.Errorf("CAPACITY_MIN_FREE_SPACE must be a non-negative size")
		}
	}
	_, _, err := ParseTemplates(cfg)
	return err
}
