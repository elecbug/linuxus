package config

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestEmbeddedEnvCoversEverySetting(t *testing.T) {
	cfg, err := ParseEnv([]byte(DefaultEnv))
	if err != nil {
		t.Fatal(err)
	}
	fields := envFields(reflect.ValueOf(&cfg).Elem(), "")
	for key := range fields {
		if !strings.Contains(DefaultEnv, "\n"+key+"=") {
			t.Errorf("embedded defaults omit %s", key)
		}
	}
	if cfg.UserService.Runtime.UID != 1000 || cfg.UserService.Runtime.GID != 1000 || fmt.Sprint(cfg.UserService.Limits.User.CPU) != "1.5" {
		t.Fatal("default runtime identity or resource limits changed")
	}
	if !cfg.Volumes.AutoEnsure || !cfg.AuthService.AllowSignup || cfg.Volumes.Host.Homes != "/var/lib/linuxus/volumes/homes" {
		t.Fatal("default disk or signup behavior changed")
	}
}

func TestEnvQuotingCommentsAndLiteralValues(t *testing.T) {
	for _, tc := range []struct{ name, input, want string }{
		{"plain", "value", "value"},
		{"unquoted hash", "value#suffix", "value#suffix"},
		{"comment", "value # comment", "value"},
		{"empty", "# comment", ""},
		{"single quoted", "' x # y=$VALUE ' # comment", " x # y=$VALUE "},
		{"double quoted", `"x # y=\"quoted\"" # comment`, `x # y="quoted"`},
		{"double escapes", `"a\nb\rc\t\\\$VALUE"`, "a\nb\rc\t\\$VALUE"},
		{"single literal", `'a\nb$VALUE'`, `a\nb$VALUE`},
		{"command text", "$(touch /tmp/not-executed)", "$(touch /tmp/not-executed)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := "\xef\xbb\xbf# defaults\r\nexport AUTH_SERVICE_SECURITY_SESSION_SECRET = " + tc.input + "\r\n"
			cfg, err := ParseEnv([]byte(data))
			if err != nil {
				t.Fatal(err)
			}
			if got := cfg.AuthService.Security.SessionSecret; got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEnvRejectsAmbiguousOrInvalidSettings(t *testing.T) {
	for _, tc := range []struct{ contents, message string }{
		{"UNKNOWN=value", "unknown setting UNKNOWN"},
		{"VOLUMES_AUTO_ENSURE=true\nVOLUMES_AUTO_ENSURE=false", "duplicate setting VOLUMES_AUTO_ENSURE"},
		{"VOLUMES_AUTO_ENSURE=yes", "expected true or false"},
		{"USER_SERVICE_RUNTIME_UID=abc", "expected an integer"},
		{"USER_SERVICE_RUNTIME_UID=999999999999999999999999", "expected an integer"},
		{"AUTH_SERVICE_SECURITY_SESSION_SECRET='private", "unterminated quoted value"},
		{"AUTH_SERVICE_SECURITY_SESSION_SECRET=\"private\"suffix", "unexpected text"},
		{"AUTH_SERVICE_SECURITY_SESSION_SECRET=private\x00suffix", "NUL is not allowed"},
		{"no assignment", "expected KEY=VALUE"},
		{"BAD-KEY=private", "expected KEY=VALUE"},
		{"# just comments\n", "contains no settings"},
	} {
		_, err := ParseEnv([]byte(tc.contents))
		if err == nil || !strings.Contains(err.Error(), tc.message) {
			t.Errorf("expected %q, got %v", tc.message, err)
		}
		if err != nil && strings.Contains(err.Error(), "private") {
			t.Error("configuration error exposed a value")
		}
	}
}

func TestEnvDoesNotInheritOrModifyProcessEnvironment(t *testing.T) {
	t.Setenv("VOLUMES_AUTO_ENSURE", "true")
	t.Setenv("AUTH_SERVICE_SECURITY_SESSION_SECRET", "process-secret")
	cfg, err := ParseEnv([]byte("AUTH_SERVICE_SECURITY_SESSION_SECRET=$AUTH_SERVICE_SECURITY_SESSION_SECRET\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Volumes.AutoEnsure || cfg.AuthService.Security.SessionSecret != "$AUTH_SERVICE_SECURITY_SESSION_SECRET" {
		t.Fatal("process environment changed file configuration")
	}
}
