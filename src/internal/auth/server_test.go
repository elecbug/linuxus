package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elecbug/linuxus/src/internal/common/config"
)

func TestAdminRouteSurvivesEnvAndContainerConfiguration(t *testing.T) {
	for _, admin := range []string{"admin", "ops/classroom", ""} {
		contents := strings.Replace(config.DefaultEnv, "AUTH_SERVICE_SERVICE_URL_ADMIN='admin'", "AUTH_SERVICE_SERVICE_URL_ADMIN='"+admin+"'", 1)
		cfg, err := config.ParseEnv([]byte(contents))
		if err != nil {
			t.Fatal(err)
		}
		cfg.AuthService.Mounts.ContainerAuthListPath = filepath.Join(t.TempDir(), "AUTH_LIST")
		if err := os.WriteFile(cfg.AuthService.Mounts.ContainerAuthListPath, []byte("admin:hash\n"), 0600); err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(cfg)
		if err != nil {
			t.Fatal(err)
		}
		t.Setenv("ENV", string(data))
		app, err := parseConfig()
		if err != nil {
			t.Fatal(err)
		}
		want := admin
		if want == "" {
			want = "admin"
		}
		if app.AdminPath != want {
			t.Fatalf("route lost in configuration transport: %q", app.AdminPath)
		}
	}
}

func TestInvalidAdminRouteRejectedBeforeLoadingAccounts(t *testing.T) {
	cfg, err := config.ParseEnv([]byte(config.DefaultEnv))
	if err != nil {
		t.Fatal(err)
	}
	cfg.AuthService.ServiceURL.Admin = "service/admin"
	cfg.AuthService.Mounts.ContainerAuthListPath = filepath.Join(t.TempDir(), "missing")
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("ENV", string(data))
	if _, err := parseConfig(); err == nil || !strings.Contains(err.Error(), "administrator route") {
		t.Fatalf("invalid route was not rejected before runtime access: %v", err)
	}
}
