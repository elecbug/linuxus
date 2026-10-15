package user

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func passwordFixture(t *testing.T) (string, string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "AUTH_LIST")
	if err := os.WriteFile(path, []byte("# classroom accounts\nalice:"+string(hash)+"\nbob:unchanged\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := UpdateAccount(path, "alice", "class", "class-a"); err != nil {
		t.Fatal(err)
	}
	users, err := LoadUsers(path)
	if err != nil {
		t.Fatal(err)
	}
	return path, users["alice"]
}

func TestChangePasswordPreservesIdentityAndMetadata(t *testing.T) {
	path, record := passwordFixture(t)
	before, _ := os.Stat(path)
	if err := ChangePassword(path, "alice", record, "old-password", "new-password"); err != nil {
		t.Fatal(err)
	}
	after, _ := os.Stat(path)
	users, _ := LoadUsers(path)
	account, err := ParseAccount(users["alice"])
	if err != nil {
		t.Fatal(err)
	}
	previous, _ := ParseAccount(record)
	if !os.SameFile(before, after) || users["bob"] != "unchanged" || account.Class != "class-a" || account.Generation == previous.Generation {
		t.Fatal("password change lost account metadata, unrelated data, or the bind inode")
	}
	if CheckPassword(users["alice"], "new-password") != nil || CheckPassword(users["alice"], "old-password") == nil {
		t.Fatal("password change did not replace the credential")
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "# classroom accounts\n") {
		t.Fatal("password change removed comments")
	}
}

func TestPasswordChangeRejectsWrongPasswordAndConcurrentAccountUpdates(t *testing.T) {
	for _, action := range []string{"wrong-password", "lock", "password", "maintenance", "remove", "replace"} {
		t.Run(action, func(t *testing.T) {
			path, record := passwordFixture(t)
			password, want := "old-password", ErrAccountChanged
			switch action {
			case "wrong-password":
				password, want = "wrong", ErrPasswordMismatch
			case "remove", "replace":
				if err := RemoveUser(path, map[string]string{}, "alice"); err != nil {
					t.Fatal(err)
				}
				if action == "replace" {
					if err := AddUser(path, map[string]string{}, "alice", "old-password"); err != nil {
						t.Fatal(err)
					}
				}
			default:
				if err := UpdateAccount(path, "alice", action, "operator-password"); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := os.ReadFile(path)
			if err := ChangePassword(path, "alice", record, password, "new-password"); !errors.Is(err, want) {
				t.Fatalf("error=%v want=%v", err, want)
			}
			after, _ := os.ReadFile(path)
			if string(before) != string(after) {
				t.Fatal("rejected password change modified the file")
			}
		})
	}
}

func TestConcurrentPasswordChangesCannotBothCommit(t *testing.T) {
	path, record := passwordFixture(t)
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, password := range []string{"first-password", "second-password"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- ChangePassword(path, "alice", record, "old-password", password)
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	successful, rejected := 0, 0
	for err := range results {
		if err == nil {
			successful++
		} else if errors.Is(err, ErrAccountChanged) {
			rejected++
		} else {
			t.Fatal(err)
		}
	}
	if successful != 1 || rejected != 1 {
		t.Fatalf("committed=%d rejected=%d", successful, rejected)
	}
}
