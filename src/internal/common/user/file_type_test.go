package user

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestRejectNonregularAuthLists(t *testing.T) {
	root := t.TempDir()
	pipe := filepath.Join(root, "pipe")
	if err := syscall.Mkfifo(pipe, 0600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(pipe, alias); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{root, pipe, alias, "/dev/null"} {
		if _, err := LoadUsers(path); err == nil {
			t.Fatalf("read nonregular auth list %s", path)
		}
		if err := EnsureFile(path); err == nil {
			t.Fatalf("initialized nonregular auth list %s", path)
		}
	}
}
