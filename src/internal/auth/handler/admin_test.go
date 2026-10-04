package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/elecbug/linuxus/src/internal/common/user"
)

func adminFixture(t *testing.T) (*App, *http.Cookie, *http.Cookie) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "AUTH_LIST")
	os.WriteFile(path, []byte("admin:admin-hash\nalice:alice-hash\n"), 0600)
	a := NewApp(&AppConfig{AuthListFile: path, AdminID: "admin", SessionKey: []byte("test-secret"), LoginPath: "login", LogoutPath: "logout", ServicePath: "service", TerminalPath: "terminal", SignupPath: "signup"})
	t.Cleanup(a.Stop)
	a.RegisterRoutes()
	cookies := []*http.Cookie{}
	for _, id := range []string{"admin", "alice"} {
		w := httptest.NewRecorder()
		a.setSessionCookie(w, httptest.NewRequest("POST", "/login", nil), id, id+"-hash")
		cookies = append(cookies, w.Result().Cookies()[0])
	}
	return a, cookies[0], cookies[1]
}

func TestAdminRequiresPrivilegeAndCSRF(t *testing.T) {
	a, admin, alice := adminFixture(t)
	for _, tc := range []struct {
		cookie        *http.Cookie
		method, token string
		want          int
	}{{nil, "GET", "", 401}, {alice, "GET", "", 403}, {admin, "GET", "", 200}, {admin, "POST", "", 403}, {admin, "POST", "wrong", 403}} {
		r := httptest.NewRequest(tc.method, "/admin", strings.NewReader(`{"user_id":"alice","action":"lock"}`))
		if tc.cookie != nil {
			r.AddCookie(tc.cookie)
		}
		r.Header.Set("X-CSRF-Token", tc.token)
		w := httptest.NewRecorder()
		if tc.method == "GET" {
			a.handleAdmin(w, r)
		} else {
			a.handleAdminUsers(w, r)
		}
		if w.Code != tc.want {
			t.Fatalf("status=%d want=%d body=%s", w.Code, tc.want, w.Body.String())
		}
	}
	users, _ := user.LoadUsers(a.authListFile)
	if user.IsLocked(users["alice"]) {
		t.Fatal("unauthorized request mutated account")
	}
}

func TestLockUnlockNeverRevivesOldCookies(t *testing.T) {
	a, _, cookie := adminFixture(t)
	r := httptest.NewRequest("GET", "/terminal/", nil)
	r.AddCookie(cookie)
	for _, operation := range []string{"lock", "unlock"} {
		if err := user.UpdateAccount(a.authListFile, "alice", operation, ""); err != nil {
			t.Fatal(err)
		}
		if _, ok := a.getSessionID(r); ok {
			t.Fatalf("%s revived a previous session", operation)
		}
	}
}

func TestSessionSnapshotCancelsRevokedConnections(t *testing.T) {
	a, _, cookie := adminFixture(t)
	a.managerBaseURL = ""
	r := httptest.NewRequest("GET", "/terminal/", nil)
	r.AddCookie(cookie)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a.sessionCancels[r] = cancel
	a.activeSessions["alice"] = 1
	user.UpdateAccount(a.authListFile, "alice", "lock", "")
	if err := a.reportAllSessions(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("revoked connection stayed open")
	}
}

func TestAdminLockChangesAccountAndStopsRuntimeWithoutExposingCredentials(t *testing.T) {
	a, cookie, _ := adminFixture(t)
	calls := 0
	manager := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("X-Manager-Session-Secret") != "manager-secret" {
			t.Error("missing Manager credential")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer manager.Close()
	a.managerBaseURL = manager.URL
	a.managerSessionSecret = "manager-secret"
	r := httptest.NewRequest("POST", "/admin/api/users", strings.NewReader(`{"user_id":"alice","action":"lock"}`))
	r.AddCookie(cookie)
	r.Header.Set("X-CSRF-Token", a.adminToken(r))
	w := httptest.NewRecorder()
	a.handleAdminUsers(w, r)
	if w.Code != 200 || calls != 1 {
		t.Fatalf("lock failed: %d %s", w.Code, w.Body.String())
	}
	users, _ := user.LoadUsers(a.authListFile)
	if !user.IsLocked(users["alice"]) {
		t.Fatal("account not locked")
	}
	list := httptest.NewRequest("GET", "/admin/api/users", nil)
	list.AddCookie(cookie)
	w = httptest.NewRecorder()
	a.handleAdminUsers(w, list)
	if w.Code != 200 || strings.Contains(w.Body.String(), "admin-hash") || strings.Contains(w.Body.String(), "manager-secret") || strings.Contains(w.Body.String(), "alice-hash") {
		t.Fatal("account listing leaked credentials or failed")
	}
}

func TestAdminKeepsPartialStatusAndShowsDiskWarning(t *testing.T) {
	a, cookie, _ := adminFixture(t)
	manager := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Linuxus-Disk-Error", "Disk usage is unavailable; check the host disk service.")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"alice":{"state":"running","sessions":2}}`))
	}))
	defer manager.Close()
	a.managerBaseURL = manager.URL
	a.managerClient = manager.Client()
	r := httptest.NewRequest(http.MethodGet, "/admin/api/users", nil)
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	a.handleAdminUsers(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"warning":"Disk usage is unavailable`) || !strings.Contains(w.Body.String(), `"state":"running"`) {
		t.Fatalf("partial status or warning lost: %d %s", w.Code, w.Body)
	}
}
