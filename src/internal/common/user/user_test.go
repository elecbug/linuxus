package user

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestSyncUsersReplacesChangedCredentialsWithSameCount(t *testing.T) {
	p := filepath.Join(t.TempDir(), "AUTH_LIST")
	if err := os.WriteFile(p, []byte("alice:new-hash\ncarol:carol-hash\n"), 0600); err != nil {
		t.Fatal(err)
	}
	users := map[string]string{"alice": "old-hash", "bob": "bob-hash"}
	if err := SyncUsers(users, p); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(users, map[string]string{"alice": "new-hash", "carol": "carol-hash"}) {
		t.Fatal(users)
	}
}

func TestInitializeAuthFileAndAppendWithoutTrailingNewline(t *testing.T) {
	p := filepath.Join(t.TempDir(), "data", "AUTH_LIST")
	if err := EnsureFile(p); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode=%v", info.Mode())
	}
	if err := os.WriteFile(p, []byte("alice:existing-hash"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureFile(p); err != nil {
		t.Fatal(err)
	}
	users, err := LoadUsers(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := AddUser(p, users, "bob", "password"); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadUsers(p)
	if err != nil {
		t.Fatal(err)
	}
	if loaded["alice"] != "existing-hash" || len(loaded) != 2 {
		t.Fatal("existing credentials were lost")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(loaded["bob"]), []byte("password")); err != nil {
		t.Fatal(err)
	}
}

func TestFailedCredentialWritesLeaveMemoryUnchanged(t *testing.T) {
	p := t.TempDir() // A directory cannot be opened as an auth file.
	users := map[string]string{"alice": "hash"}
	if err := AddUser(p, users, "bob", "password"); err == nil {
		t.Fatal("expected write failure")
	}
	if err := RemoveUser(p, users, "alice"); err == nil {
		t.Fatal("expected write failure")
	}
	if !reflect.DeepEqual(users, map[string]string{"alice": "hash"}) {
		t.Fatal("memory diverged from file")
	}
	if err := AddUser(filepath.Join(p, "AUTH_LIST"), users, "../outside", "password"); err == nil {
		t.Fatal("accepted path traversal ID")
	}
}

func TestStaleUserMapCannotCreateDuplicateAccount(t *testing.T) {
	path := filepath.Join(t.TempDir(), "AUTH_LIST")
	if err := os.WriteFile(path, []byte("alice:original-hash\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := AddUser(path, map[string]string{}, "alice", "replacement"); err == nil {
		t.Fatal("stale map overwrote an existing account")
	}
	loaded, err := LoadUsers(path)
	if err != nil || loaded["alice"] != "original-hash" {
		t.Fatal("existing credentials changed")
	}
}

func TestCredentialErrorsDoNotExposeHashes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "AUTH_LIST")
	if err := os.WriteFile(path, []byte("bad/id:private-password-hash\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadUsers(path)
	if err == nil || strings.Contains(err.Error(), "private-password-hash") {
		t.Fatalf("unsafe parse error: %v", err)
	}
}

func TestCredentialReadersHonorWriterLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "AUTH_LIST")
	if err := os.WriteFile(path, []byte("alice:hash\n"), 0600); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	done := make(chan error, 1)
	go func() { _, err := LoadUsers(path); done <- err }()
	select {
	case err := <-done:
		t.Fatalf("reader ignored an active file writer: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("reader did not resume")
	}
}

func TestIndependentWritersCannotRegisterSameID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "AUTH_LIST")
	if err := EnsureFile(path); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var successes atomic.Int32
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if AddUser(path, map[string]string{}, "alice", "password") == nil {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("successful duplicate registrations: %d", successes.Load())
	}
	users, err := LoadUsers(path)
	if err != nil || len(users) != 1 {
		t.Fatalf("invalid auth list: %v", err)
	}
}

func TestRemoveUsesCurrentFileAndPreservesBindMountInode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "AUTH_LIST")
	if err := os.WriteFile(path, []byte("# users\nalice:hash\nbob:hash\n"), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	stale := map[string]string{}
	if err := RemoveUser(path, stale, "alice"); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Fatal("auth file replacement broke the bind mount")
	}
	if !reflect.DeepEqual(stale, map[string]string{"bob": "hash"}) {
		t.Fatal("did not refresh the caller snapshot")
	}
	loaded, err := LoadUsers(path)
	if err != nil || !reflect.DeepEqual(loaded, stale) {
		t.Fatalf("file differs from snapshot: %v", err)
	}
}
