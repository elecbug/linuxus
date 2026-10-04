package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/elecbug/linuxus/src/internal/common/user"
)

func TestConcurrentSignupAndLoginReloads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "AUTH_LIST")
	if err := user.EnsureFile(path); err != nil {
		t.Fatal(err)
	}
	a := NewApp(&AppConfig{
		Users: map[string]string{}, AuthListFile: path, SessionKey: []byte("test-secret"),
		LoginPath: "login", LogoutPath: "logout", ServicePath: "service", TerminalPath: "terminal", SignupPath: "signup", AllowSignup: true,
	})
	defer a.Stop()
	a.RegisterRoutes()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			form := url.Values{"id": {fmt.Sprintf("user%d", i)}, "password": {"password"}, "confirm_password": {"password"}}
			req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rr := httptest.NewRecorder()
			a.mux.ServeHTTP(rr, req)
			if rr.Code != http.StatusSeeOther {
				t.Errorf("signup failed: %d", rr.Code)
			}
		}(i)
		go func() {
			defer wg.Done()
			for j := 0; j < 4; j++ {
				form := url.Values{"id": {"unknown"}, "password": {"wrong"}}
				req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				rr := httptest.NewRecorder()
				a.mux.ServeHTTP(rr, req)
				if rr.Code != http.StatusOK {
					t.Errorf("login failed: %d", rr.Code)
				}
			}
		}()
	}
	wg.Wait()
	loaded, err := user.LoadUsers(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 4 {
		t.Fatalf("lost signups: %d", len(loaded))
	}
}
