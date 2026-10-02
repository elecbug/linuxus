package user

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

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
