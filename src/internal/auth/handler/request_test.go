package handler

import (
	"io"
	"net"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestClientIPIgnoresForgedForwardedPrefix(t *testing.T) {
	_, trusted, _ := net.ParseCIDR("10.0.0.0/8")
	a := &App{trustedProxies: []*net.IPNet{trusted}}
	for _, tc := range []struct{ remote, forwarded, want string }{
		{"10.0.0.1:8080", "192.0.2.99, 203.0.113.10", "203.0.113.10"},
		{"10.0.0.1:8080", "192.0.2.99, 203.0.113.10, 10.0.0.2", "203.0.113.10"},
		{"203.0.113.10:8080", "192.0.2.99", "203.0.113.10"},
		{"10.0.0.1:8080", "not-an-ip", "10.0.0.1"},
	} {
		r := httptest.NewRequest("GET", "/login", nil)
		r.RemoteAddr = tc.remote
		r.Header.Set("X-Forwarded-For", tc.forwarded)
		if got := a.clientIP(r); got != tc.want {
			t.Fatalf("client IP=%q, want %q", got, tc.want)
		}
	}
}

type delayedForm struct {
	reader           io.Reader
	once             sync.Once
	reading, release chan struct{}
}

func (b *delayedForm) Read(p []byte) (int, error) {
	b.once.Do(func() { close(b.reading) })
	<-b.release
	return b.reader.Read(p)
}
func (b *delayedForm) Close() error { return nil }

func TestSlowSignupDoesNotHoldCredentialMutex(t *testing.T) {
	path := filepath.Join(t.TempDir(), "AUTH_LIST")
	if err := os.WriteFile(path, []byte("alice:hash\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a := NewApp(&AppConfig{Users: map[string]string{}, AuthListFile: path, AllowSignup: true, LoginPath: "login", SignupPath: "signup"})
	defer a.Stop()
	body := &delayedForm{reader: strings.NewReader("id=alice&password=password&confirm_password=password"), reading: make(chan struct{}), release: make(chan struct{})}
	var once sync.Once
	release := func() { once.Do(func() { close(body.release) }) }
	defer release()
	r := httptest.NewRequest("POST", "/signup", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	done := make(chan struct{})
	go func() { a.handleSignup(httptest.NewRecorder(), r); close(done) }()
	select {
	case <-body.reading:
	case <-time.After(time.Second):
		t.Fatal("signup body was not read")
	}
	if !a.usersMu.TryLock() {
		t.Error("slow request blocked credential access")
	} else {
		a.usersMu.Unlock()
	}
	release()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("signup did not finish")
	}
}
