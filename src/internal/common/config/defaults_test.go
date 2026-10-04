package config

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func TestInitCreatesPrivateConfigAndPreservesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := InitFile(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != DefaultEnv {
		t.Fatal("init did not copy embedded defaults exactly")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("configuration mode=%v", info.Mode().Perm())
	}
	if err := os.WriteFile(path, []byte("local settings"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := InitFile(path); err == nil {
		t.Fatal("init overwrote an existing config")
	}
	data, err = os.ReadFile(path)
	if err != nil || string(data) != "local settings" {
		t.Fatal("existing settings changed")
	}
}

func TestInitDoesNotFollowExistingSymlinks(t *testing.T) {
	for _, targetExists := range []bool{false, true} {
		root := t.TempDir()
		target := filepath.Join(root, "target")
		if targetExists {
			if err := os.WriteFile(target, []byte("preserve"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		path := filepath.Join(root, ".env")
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
		if err := InitFile(path); err == nil {
			t.Fatal("init followed an existing symlink")
		}
		if targetExists {
			data, err := os.ReadFile(target)
			if err != nil || string(data) != "preserve" {
				t.Fatal("symlink target changed")
			}
		} else if _, err := os.Stat(target); !os.IsNotExist(err) {
			t.Fatal("init created a symlink target")
		}
	}
}

func TestConcurrentInitCreatesOneCompleteFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	var wg sync.WaitGroup
	var successes atomic.Int32
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if InitFile(path) == nil {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("successful initializations=%d", successes.Load())
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != DefaultEnv {
		t.Fatal("concurrent init damaged defaults")
	}
}
