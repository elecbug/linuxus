package handler

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestTerminalProxyHidesSessionAndPreservesRequest(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := r.Cookie("session"); err != http.ErrNoCookie {
			t.Error("Auth credential reached user runtime")
		}
		if cookie, err := r.Cookie("terminal-setting"); err != nil || cookie.Value != "dark" {
			t.Error("terminal cookie was lost")
		}
		if r.URL.RequestURI() != "/socket?arg=a%2Fb" {
			t.Errorf("backend URI: %s", r.URL.RequestURI())
		}
		if r.Header.Get("X-Forwarded-Proto") != "https" || r.Header.Get("X-Forwarded-Host") != "linuxus.example" {
			t.Error("external origin was not preserved")
		}
		if r.Header.Get("X-Forwarded-For") != "203.0.113.8" {
			t.Error("spoofed forwarding header reached runtime")
		}
		io.WriteString(w, "backend response")
	}))
	defer backend.Close()
	target, _ := url.Parse(backend.URL)
	a := &App{terminalPath: "terminal"}
	r := httptest.NewRequest("GET", "https://linuxus.example/terminal/socket?arg=a%2Fb", nil)
	r.RemoteAddr = "203.0.113.8:1234"
	r.Header.Set("X-Forwarded-For", "192.0.2.1")
	r.Header.Set("X-Forwarded-Proto", "http")
	r.AddCookie(&http.Cookie{Name: "session", Value: "private"})
	r.AddCookie(&http.Cookie{Name: "terminal-setting", Value: "dark"})
	w := httptest.NewRecorder()
	a.terminalProxy(target).ServeHTTP(w, r)
	if w.Code != http.StatusOK || w.Body.String() != "backend response" {
		t.Fatalf("proxy response: %d %s", w.Code, w.Body.String())
	}
	if _, err := r.Cookie("session"); err != nil {
		t.Error("proxy mutated the original request")
	}
}

func TestSecureCookieRespectsTrustedScheme(t *testing.T) {
	_, trusted, _ := net.ParseCIDR("10.0.0.0/8")
	a := &App{sessionKey: []byte("secret"), trustedProxies: []*net.IPNet{trusted}}
	for _, tc := range []struct {
		name, target, remote string
		forwarded            []string
		secure               bool
	}{
		{"direct HTTP", "http://example/login", "203.0.113.8:1234", nil, false},
		{"direct HTTPS", "https://example/login", "203.0.113.8:1234", []string{"http"}, true},
		{"trusted TLS proxy", "http://example/login", "10.0.0.1:1234", []string{"https"}, true},
		{"untrusted header", "http://example/login", "203.0.113.8:1234", []string{"https"}, false},
		{"ambiguous list", "http://example/login", "10.0.0.1:1234", []string{"https, http"}, false},
		{"repeated header", "http://example/login", "10.0.0.1:1234", []string{"https", "http"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", tc.target, nil)
			r.RemoteAddr = tc.remote
			for _, v := range tc.forwarded {
				r.Header.Add("X-Forwarded-Proto", v)
			}
			w := httptest.NewRecorder()
			a.setSessionCookie(w, r, "alice", "hash")
			if got := w.Result().Cookies()[0].Secure; got != tc.secure {
				t.Fatalf("Secure=%t, want %t", got, tc.secure)
			}
		})
	}
}

func TestWebSocketRequiresCompleteUpgradeToken(t *testing.T) {
	for _, tc := range []struct {
		connection []string
		upgrade    string
		want       bool
	}{
		{[]string{"keep-alive, Upgrade"}, "websocket", true},
		{[]string{"keep-alive", "UPGRADE"}, "WebSocket", true},
		{[]string{"notupgrade"}, "websocket", false},
		{[]string{"upgrade"}, "h2c", false},
	} {
		r := httptest.NewRequest("GET", "/terminal/", nil)
		for _, v := range tc.connection {
			r.Header.Add("Connection", v)
		}
		r.Header.Set("Upgrade", tc.upgrade)
		if got := isWebSocketRequest(r); got != tc.want {
			t.Errorf("headers %v/%s: got %t", tc.connection, tc.upgrade, got)
		}
	}
}

func TestTerminalProxyPreservesBidirectionalUpgrade(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isWebSocketRequest(r) || r.URL.Path != "/ws" {
			t.Errorf("upgrade request was changed: %s", r.URL)
		}
		if _, err := r.Cookie("session"); err != http.ErrNoCookie {
			t.Error("session leaked on upgrade")
		}
		conn, rw, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(3 * time.Second))
		fmt.Fprint(rw, "HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n\r\n")
		if err := rw.Flush(); err != nil {
			t.Error(err)
			return
		}
		data := make([]byte, 3)
		if _, err := io.ReadFull(rw, data); err != nil {
			t.Error(err)
			return
		}
		if _, err := rw.Write(data); err != nil {
			t.Error(err)
			return
		}
		if err := rw.Flush(); err != nil {
			t.Error(err)
		}
	}))
	defer backend.Close()
	target, _ := url.Parse(backend.URL)
	a := &App{terminalPath: "terminal"}
	front := httptest.NewServer(a.terminalProxy(target))
	defer front.Close()
	frontURL, _ := url.Parse(front.URL)
	conn, err := net.DialTimeout("tcp", frontURL.Host, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := fmt.Fprint(conn, "GET /terminal/ws HTTP/1.1\r\nHost: example\r\nConnection: Upgrade\r\nUpgrade: websocket\r\nCookie: session=private\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(conn)
	response, err := http.ReadResponse(reader, &http.Request{Method: "GET"})
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("upgrade status=%d", response.StatusCode)
	}
	if _, err := io.WriteString(conn, "abc"); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 3)
	if _, err := io.ReadFull(reader, data); err != nil {
		t.Fatal(err)
	}
	if string(data) != "abc" {
		t.Fatalf("upgrade bytes changed: %q", data)
	}
}
