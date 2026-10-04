package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfigLocationIgnoresWorkingDirectory(t *testing.T) {
	old, exists := os.LookupEnv(ConfigFileEnv)
	if err := os.Unsetenv(ConfigFileEnv); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if exists {
			os.Setenv(ConfigFileEnv, old)
		} else {
			os.Unsetenv(ConfigFileEnv)
		}
	})
	for range 2 {
		root := t.TempDir()
		t.Chdir(root)
		if err := os.WriteFile(".env", []byte("VOLUMES_AUTO_ENSURE=false\n"), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := ResolveConfigFile()
		if err != nil || got != DefaultConfigFile {
			t.Fatalf("config=%q error=%v", got, err)
		}
	}
}

func TestConfigOverrideRequiresAbsolutePath(t *testing.T) {
	for _, path := range []string{"", ".env", "../.env"} {
		t.Setenv(ConfigFileEnv, path)
		if _, err := ResolveConfigFile(); err == nil {
			t.Fatalf("accepted relative override %q", path)
		}
	}
	absolute := filepath.Join(t.TempDir(), "config", "..", ".env")
	t.Setenv(ConfigFileEnv, absolute)
	got, err := ResolveConfigFile()
	if err != nil || got != filepath.Clean(absolute) {
		t.Fatalf("config=%q error=%v", got, err)
	}
}

func TestChildConfigReplacesInheritedOverrides(t *testing.T) {
	original := []string{"LANG=C", "LINUXUS_CONFIG=/stale/.env", "LINUXUS_CONFIG=/other/.env", "KEEP=value"}
	got := WithConfigFile(original, "/chosen/.env")
	want := []string{"LANG=C", "KEEP=value", "LINUXUS_CONFIG=/chosen/.env"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("child env=%v", got)
	}
	if original[1] != "LINUXUS_CONFIG=/stale/.env" {
		t.Fatal("parent environment mutated")
	}
}
