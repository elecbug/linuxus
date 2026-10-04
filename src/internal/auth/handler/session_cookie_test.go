package handler

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSessionExpiryCredentialsAndSignature(t *testing.T) {
	path := filepath.Join(t.TempDir(), "AUTH_LIST")
	if err := os.WriteFile(path, []byte("alice:original-hash\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a := &App{authListFile: path, sessionKey: []byte("test-secret")}
	w := httptest.NewRecorder()
	a.setSessionCookie(w, httptest.NewRequest("POST", "/login", nil), "alice", "original-hash")
	cookie := w.Result().Cookies()[0]
	check := func(cookie *http.Cookie, want bool) {
		t.Helper()
		r := httptest.NewRequest("GET", "/terminal/", nil)
		r.AddCookie(cookie)
		id, ok := a.getSessionID(r)
		if ok != want || (ok && id != "alice") {
			t.Fatalf("session accepted=%t, want %t", ok, want)
		}
	}
	check(cookie, true)
	expired := "alice|" + strconv.FormatInt(time.Now().Add(-time.Minute).Unix(), 10)
	check(&http.Cookie{Name: "session", Value: base64.StdEncoding.EncodeToString([]byte(expired + "|" + a.sign("session|"+expired+"|original-hash")))}, false)
	raw, _ := base64.StdEncoding.DecodeString(cookie.Value)
	tampered := strings.Replace(string(raw), "alice", "bob", 1)
	check(&http.Cookie{Name: "session", Value: base64.StdEncoding.EncodeToString([]byte(tampered))}, false)
	// Legacy cookies do not contain a verifiable expiry and require re-login.
	check(&http.Cookie{Name: "session", Value: base64.StdEncoding.EncodeToString([]byte("alice|" + a.sign("alice")))}, false)
	for _, contents := range []string{"", "alice:changed-hash\n"} {
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
		check(cookie, false)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	check(cookie, false)
}
