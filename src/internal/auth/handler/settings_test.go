package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/elecbug/linuxus/src/internal/common/user"
	"golang.org/x/crypto/bcrypt"
)

func settingsFixture(t *testing.T) (*App, *http.Cookie) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "AUTH_LIST")
	hash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("alice:"+string(hash)+"\nbob:untouched\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a := NewApp(&AppConfig{AuthListFile: path, AdminID: "admin", SessionKey: []byte("test-secret"), SettingsPath: "account/preferences", LoginPath: "auth/login", LogoutPath: "auth/logout", ServicePath: "workspace", TerminalPath: "shell", SignupPath: "auth/signup"})
	t.Cleanup(a.Stop)
	a.RegisterRoutes()
	w := httptest.NewRecorder()
	a.setSessionCookie(w, httptest.NewRequest("POST", "/auth/login", nil), "alice", string(hash))
	return a, w.Result().Cookies()[0]
}

func settingsRequest(a *App, cookie *http.Cookie, values url.Values) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/account/preferences", strings.NewReader(values.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Accept", "application/json")
	if cookie != nil {
		r.AddCookie(cookie)
	}
	return r
}

func passwordValues(a *App, cookie *http.Cookie) url.Values {
	r := httptest.NewRequest(http.MethodGet, "/account/preferences", nil)
	r.AddCookie(cookie)
	return url.Values{"csrf_token": {a.settingsToken(r)}, "current_password": {"old-password"}, "new_password": {"new-password"}, "confirm_password": {"new-password"}}
}

func TestPersonalSettingsRoutesRequireSessionButNotAdmin(t *testing.T) {
	a, cookie := settingsFixture(t)
	for _, tc := range []struct {
		path, method string
		cookie       *http.Cookie
		want         int
	}{
		{"/account/preferences", "GET", cookie, 200}, {"/account/preferences/", "GET", cookie, 200},
		{"/account/preferences", "GET", nil, 303}, {"/account/preferences/missing", "GET", cookie, 404},
		{"/account/preferences", "DELETE", cookie, 405},
	} {
		r := httptest.NewRequest(tc.method, tc.path, nil)
		if tc.cookie != nil {
			r.AddCookie(tc.cookie)
		}
		w := httptest.NewRecorder()
		a.mux.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("%s %s: %d %s", tc.method, tc.path, w.Code, w.Body)
		}
		if tc.want == 200 {
			body := w.Body.String()
			if !strings.Contains(body, `data-view="settings"`) || !strings.Contains(body, `action="/account/preferences"`) || strings.Contains(body, `src="/shell/"`) || strings.Contains(body, `id="admin-panel"`) {
				t.Fatal("personal settings routed incorrectly or started a terminal")
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("settings can be cached")
			}
		}
	}
}

func TestPersonalPasswordValidationNeverChangesCredentials(t *testing.T) {
	for _, name := range []string{"no-session", "no-csrf", "admin-token", "wrong-current", "empty", "mismatch", "utf8-length", "oversized", "query-password"} {
		t.Run(name, func(t *testing.T) {
			a, cookie := settingsFixture(t)
			values := passwordValues(a, cookie)
			want := 400
			switch name {
			case "no-session":
				cookie = nil
				want = 401
			case "no-csrf":
				values.Del("csrf_token")
				want = 403
			case "admin-token":
				r := httptest.NewRequest("GET", "/account/preferences", nil)
				r.AddCookie(cookie)
				values.Set("csrf_token", a.adminToken(r))
				want = 403
			case "wrong-current":
				values.Set("current_password", "wrong")
			case "empty":
				values.Set("new_password", "")
				values.Set("confirm_password", "")
			case "mismatch":
				values.Set("confirm_password", "another-password")
			case "utf8-length":
				values.Set("new_password", strings.Repeat("é", 37))
				values.Set("confirm_password", strings.Repeat("é", 37))
			case "oversized":
				values.Set("padding", strings.Repeat("x", 17<<10))
			case "query-password":
				values.Del("current_password")
			}
			before, _ := os.ReadFile(a.authListFile)
			r := settingsRequest(a, cookie, values)
			if name == "query-password" {
				r.URL.RawQuery = "current_password=old-password"
			}
			w := httptest.NewRecorder()
			a.mux.ServeHTTP(w, r)
			if w.Code != want {
				t.Fatalf("status=%d want=%d body=%s", w.Code, want, w.Body)
			}
			after, _ := os.ReadFile(a.authListFile)
			if string(before) != string(after) || strings.Contains(w.Body.String(), "new-password") || strings.Contains(w.Body.String(), "$2a$") {
				t.Fatal("validation mutated or exposed credentials")
			}
		})
	}
}

func TestPersonalPasswordChangeRevokesSessionsAndIgnoresTargetInput(t *testing.T) {
	a, cookie := settingsFixture(t)
	active := httptest.NewRequest("GET", "/shell/", nil)
	active.AddCookie(cookie)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a.sessionCancels[active] = cancel
	otherCtx, otherCancel := context.WithCancel(context.Background())
	defer otherCancel()
	// A separate valid account proves that revocation is account-specific.
	users, _ := user.LoadUsers(a.authListFile)
	otherW := httptest.NewRecorder()
	a.setSessionCookie(otherW, active, "bob", users["bob"])
	other := httptest.NewRequest("GET", "/shell/", nil)
	other.AddCookie(otherW.Result().Cookies()[0])
	a.sessionCancels[other] = otherCancel
	values := passwordValues(a, cookie)
	values.Set("user_id", "bob")
	r := settingsRequest(a, cookie, values)
	w := httptest.NewRecorder()
	a.mux.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `/auth/login?password=changed`) {
		t.Fatalf("password change failed: %d %s", w.Code, w.Body)
	}
	users, _ = user.LoadUsers(a.authListFile)
	if user.CheckPassword(users["alice"], "new-password") != nil || users["bob"] != "untouched" {
		t.Fatal("password changed the wrong account")
	}
	if _, ok := a.getSessionID(active); ok {
		t.Fatal("old session is still valid")
	}
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("old terminal remained open")
	}
	select {
	case <-otherCtx.Done():
		t.Fatal("another account was disconnected")
	default:
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge != -1 || !cookies[0].HttpOnly {
		t.Fatal("current browser cookie was not cleared")
	}
	for _, tc := range []struct {
		password string
		want     int
	}{{"old-password", 200}, {"new-password", 303}} {
		form := url.Values{"id": {"alice"}, "password": {tc.password}}
		login := httptest.NewRequest("POST", "/auth/login", strings.NewReader(form.Encode()))
		login.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		out := httptest.NewRecorder()
		a.mux.ServeHTTP(out, login)
		if out.Code != tc.want {
			t.Fatalf("login after password change: %d want %d", out.Code, tc.want)
		}
		if tc.want == 303 {
			check := httptest.NewRequest("GET", "/workspace/", nil)
			check.AddCookie(out.Result().Cookies()[0])
			if _, ok := a.getSessionID(check); !ok {
				t.Fatal("new password did not create a valid session")
			}
		}
	}
}

func TestPasswordFormWorksWithoutJavaScriptAndRateLimitsFailures(t *testing.T) {
	t.Run("form", func(t *testing.T) {
		a, cookie := settingsFixture(t)
		values := passwordValues(a, cookie)
		values.Set("current_password", "wrong")
		r := settingsRequest(a, cookie, values)
		r.Header.Del("Accept")
		w := httptest.NewRecorder()
		a.mux.ServeHTTP(w, r)
		if w.Code != 400 || !strings.Contains(w.Body.String(), "Current password is incorrect.") || strings.Contains(w.Body.String(), `value="wrong"`) {
			t.Fatal("HTML form errors were not rendered safely")
		}
		values.Set("current_password", "old-password")
		r = settingsRequest(a, cookie, values)
		r.Header.Del("Accept")
		w = httptest.NewRecorder()
		a.mux.ServeHTTP(w, r)
		if w.Code != 303 || w.Header().Get("Location") != "/auth/login?password=changed" {
			t.Fatal("successful form did not redirect to configured login")
		}
		w = httptest.NewRecorder()
		a.mux.ServeHTTP(w, httptest.NewRequest("GET", "/auth/login?password=changed", nil))
		if !strings.Contains(w.Body.String(), "Password changed. Sign in with your new password.") {
			t.Fatal("login confirmation missing")
		}
	})
	t.Run("rate limit", func(t *testing.T) {
		a, cookie := settingsFixture(t)
		values := passwordValues(a, cookie)
		values.Set("current_password", "wrong")
		for i := 0; i < 6; i++ {
			w := httptest.NewRecorder()
			a.mux.ServeHTTP(w, settingsRequest(a, cookie, values))
			if i == 5 && (w.Code != 429 || w.Header().Get("Retry-After") == "") {
				t.Fatal("current password attempts were not limited")
			}
		}
	})
}

func TestPasswordChangeRejectsTerminalWaitingForPreparation(t *testing.T) {
	for _, websocket := range []bool{false, true} {
		name := "http"
		if websocket {
			name = "websocket"
		}
		t.Run(name, func(t *testing.T) {
			a, cookie := settingsFixture(t)
			preparing, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			resume := func() { once.Do(func() { close(release) }) }
			manager := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/user/up" {
					t.Errorf("unexpected manager request %s", r.URL.Path)
					http.Error(w, "unexpected request", 500)
					return
				}
				close(preparing)
				<-release
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"ok":true,"container_name":"unreachable.invalid"}`))
			}))
			defer manager.Close()
			defer resume()
			a.managerBaseURL = manager.URL
			a.managerClient = manager.Client()
			terminal := httptest.NewRequest("GET", "/shell/", nil)
			terminal.AddCookie(cookie)
			if websocket {
				terminal.Header.Set("Connection", "Upgrade")
				terminal.Header.Set("Upgrade", "websocket")
			}
			result := httptest.NewRecorder()
			done := make(chan struct{})
			go func() { a.handleTerminalProxy(result, terminal); close(done) }()
			select {
			case <-preparing:
			case <-time.After(2 * time.Second):
				t.Fatal("terminal did not reach preparation")
			}
			changed := httptest.NewRecorder()
			a.handleSettings(changed, settingsRequest(a, cookie, passwordValues(a, cookie)))
			if changed.Code != 200 {
				t.Fatalf("password change failed: %d %s", changed.Code, changed.Body)
			}
			resume()
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("revoked terminal continued to its backend")
			}
			if result.Code != 303 || result.Header().Get("Location") != "/auth/login" {
				t.Fatalf("revoked terminal was accepted after preparation: %d %s", result.Code, result.Body)
			}
			a.sessionMu.Lock()
			pending, active := len(a.sessionCancels), a.activeSessions["alice"]
			a.sessionMu.Unlock()
			if pending != 0 || active != 0 {
				t.Fatal("rejected terminal left active session state")
			}
		})
	}
}
