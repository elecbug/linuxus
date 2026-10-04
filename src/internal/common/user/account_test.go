package user

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAccountOperationsPreserveDataAndRevokeGenerations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "AUTH_LIST")
	users := map[string]string{}
	if err := AddUser(path, users, "alice", "initial-password"); err != nil {
		t.Fatal(err)
	}
	before, _ := os.Stat(path)
	previous := users["alice"]
	for _, op := range []struct {
		action, value string
		locked        bool
	}{
		{"lock", "", true}, {"unlock", "", false}, {"password", "new-password", false}, {"template", "python", false}, {"class", "class-a", false}, {"disconnect", "", false},
	} {
		if err := UpdateAccount(path, "alice", op.action, op.value); err != nil {
			t.Fatal(err)
		}
		current, err := LoadUsers(path)
		if err != nil {
			t.Fatal(err)
		}
		if current["alice"] == previous {
			t.Fatal("operation revived a prior session generation")
		}
		previous = current["alice"]
		account, err := ParseAccount(previous)
		if err != nil {
			t.Fatal(err)
		}
		if account.Locked != op.locked {
			t.Fatal("lock state lost")
		}
		after, _ := os.Stat(path)
		if !os.SameFile(before, after) {
			t.Fatal("account update replaced Docker bind inode")
		}
	}
	if err := CheckPassword(previous, "new-password"); err != nil {
		t.Fatal(err)
	}
	if err := CheckPassword(previous, "initial-password"); err == nil {
		t.Fatal("old password still accepted")
	}
	account, _ := ParseAccount(previous)
	if account.Class != "class-a" || account.Template != "" {
		t.Fatal("class assignment did not clear direct template override")
	}
}

func TestAccountFailurePreservesOriginalAndComments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "AUTH_LIST")
	original := "# operator note\nalice:legacy\nbob:untouched\n"
	os.WriteFile(path, []byte(original), 0600)
	for _, op := range []string{"unknown", "password"} {
		if err := UpdateAccount(path, "alice", op, ""); err == nil {
			t.Fatal("accepted invalid update")
		}
	}
	data, _ := os.ReadFile(path)
	if string(data) != original {
		t.Fatal("failed update changed data")
	}
	if err := UpdateAccount(path, "alice", "lock", ""); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(path)
	if !strings.Contains(string(data), "# operator note\n") || !strings.Contains(string(data), "bob:untouched\n") {
		t.Fatal("unrelated records lost")
	}
	if !IsLocked("!linuxus1!bad-record") {
		t.Fatal("corrupt account was enabled")
	}
}

func TestMaintenanceCannotBeUnlockedOrOverlapped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "AUTH_LIST")
	os.WriteFile(path, []byte("alice:legacy\n"), 0600)
	if err := UpdateAccount(path, "alice", "maintenance", ""); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"maintenance", "unlock", "password", "class", "template"} {
		if err := UpdateAccount(path, "alice", action, "value"); err == nil {
			t.Fatalf("%s bypassed disk maintenance", action)
		}
	}
	if err := UpdateAccount(path, "alice", "maintenance-end", ""); err != nil {
		t.Fatal(err)
	}
	users, _ := LoadUsers(path)
	if IsLocked(users["alice"]) {
		t.Fatal("successful maintenance changed an unlocked account")
	}
	UpdateAccount(path, "alice", "maintenance", "")
	UpdateAccount(path, "alice", "maintenance-failed", "")
	users, _ = LoadUsers(path)
	record, _ := ParseAccount(users["alice"])
	if !record.Locked || record.Maintenance {
		t.Fatal("failed maintenance did not leave recoverable lock")
	}
}

func TestRemovalCannotDiscardMaintenanceAccount(t *testing.T) {
	path := filepath.Join(t.TempDir(), "AUTH_LIST")
	if err := os.WriteFile(path, []byte("alice:legacy\n"), 0600); err != nil {
		t.Fatal(err)
	}
	stale := map[string]string{"alice": "legacy"}
	if err := UpdateAccount(path, "alice", "maintenance", ""); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := RemoveUser(path, stale, "alice"); err == nil {
		t.Fatal("removal bypassed active maintenance")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(before) {
		t.Fatal("failed removal changed the account record")
	}
}
