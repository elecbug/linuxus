package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUsablePathDoesNotCreateParentDirectories(t *testing.T) {
	root := t.TempDir()
	if err := UsablePath(filepath.Join(root, "data", "AUTH_LIST")); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("validation created files: %v", entries)
	}
}
