package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUnifiedApplicationPreservesRoleBoundary(t *testing.T) {
	a, admin, alice := adminFixture(t, "ops/classroom")
	for _, tc := range []struct {
		name          string
		cookie        *http.Cookie
		administrator bool
	}{{"administrator", admin, true}, {"user", alice, false}} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/service/", nil)
			r.AddCookie(tc.cookie)
			w := httptest.NewRecorder()
			a.mux.ServeHTTP(w, r)
			body := w.Body.String()
			for _, text := range []string{`data-view="workspace"`, `href="/static/ui.css"`, `src="/static/workspace.js"`, `src="/terminal/"`, `href="/logout"`} {
				if w.Code != 200 || !strings.Contains(body, text) {
					t.Fatalf("missing workspace element %q: %d %s", text, w.Code, body)
				}
			}
			for _, text := range []string{`id="admin-panel"`, `data-csrf-token="`, `/ops/classroom/assets/admin.js`, `data-view-link="admin"`} {
				if strings.Contains(body, text) != tc.administrator {
					t.Fatalf("role boundary failed for %q", text)
				}
			}
			if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
				t.Fatal("session page response headers missing")
			}
			if strings.Contains(body, "admin-hash") || strings.Contains(body, "alice-hash") {
				t.Fatal("credentials leaked into the page")
			}
		})
	}
}

func TestDirectAdministrationDoesNotStartTerminal(t *testing.T) {
	a, cookie, _ := adminFixture(t, "ops/classroom")
	r := httptest.NewRequest(http.MethodGet, "/ops/classroom/", nil)
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	a.mux.ServeHTTP(w, r)
	body := w.Body.String()
	if w.Code != 200 || !strings.Contains(body, `data-view="admin"`) || !strings.Contains(body, `data-terminal-url="/terminal/"`) || strings.Contains(body, `src="/terminal/"`) {
		t.Fatalf("direct administration started a terminal or failed: %d %s", w.Code, body)
	}
	if !strings.Contains(body, `aria-current="page">Administration`) || !strings.Contains(body, `data-api-url="/ops/classroom/api/users"`) {
		t.Fatal("configured navigation or API missing")
	}
}

func TestSharedAuthenticationPagesEscapeMessagesAndUseConfiguredRoutes(t *testing.T) {
	a, _, _ := adminFixture(t)
	a.loginPath, a.signupPath, a.servicePath = "auth/login", "auth/signup", "workspace"
	a.allowSignup = true
	message := `<script>alert("unsafe")</script>`
	for _, tc := range []struct {
		name   string
		render func(http.ResponseWriter)
		want   string
		status int
	}{
		{"login", func(w http.ResponseWriter) { a.renderLogin(w, message) }, `action="/auth/login"`, 200},
		{"signup", func(w http.ResponseWriter) { a.renderSignup(w, message) }, `action="/auth/signup"`, 200},
		{"error", func(w http.ResponseWriter) { a.renderError(w, message, 503) }, `href="/auth/login"`, 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tc.render(w)
			body := w.Body.String()
			if w.Code != tc.status || !strings.Contains(body, tc.want) || !strings.Contains(body, `href="/static/ui.css"`) {
				t.Fatalf("shared page failed: %d %s", w.Code, body)
			}
			if strings.Contains(body, message) || !strings.Contains(body, "&lt;script&gt;") {
				t.Fatal("error message was not escaped")
			}
			if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
				t.Fatal("authentication page response headers missing")
			}
		})
	}
}
