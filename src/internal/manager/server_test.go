package manager

import (
	"encoding/json"
	"testing"

	ctlconfig "github.com/elecbug/linuxus/src/internal/common/config"
	"github.com/elecbug/linuxus/src/internal/common/diskservice"
)

func TestAutoEnsureConfigFromEnv(t *testing.T) {
	for _, tc := range []struct {
		name      string
		enabled   bool
		socket    string
		wantError bool
	}{
		{"enabled", true, diskservice.ContainerSocket, false},
		{"disabled", false, "", false},
		{"missing_socket", true, "", true},
		{"relative_socket", true, "disk.sock", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := ctlconfig.ParseEnv([]byte(ctlconfig.DefaultEnv))
			if err != nil {
				t.Fatal(err)
			}
			cfg.Volumes.AutoEnsure = tc.enabled
			env, err := json.Marshal(cfg)
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("ENV", string(env))
			t.Setenv("USER_IMAGE", "test-user")
			t.Setenv("DISK_SERVICE_SOCKET", tc.socket)
			parsed, err := parseConfigFromEnv()
			if (err != nil) != tc.wantError {
				t.Fatalf("config=%+v error=%v", parsed, err)
			}
			if !tc.wantError && (parsed.AutoEnsure != tc.enabled || parsed.DiskServiceSocket != tc.socket) {
				t.Fatalf("lost setting: %+v", parsed)
			}
		})
	}
}
