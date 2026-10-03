package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/elecbug/linuxus/src/internal/common/ruleset"
	"github.com/elecbug/linuxus/src/internal/common/user"
)

const sessionLifetime = 12 * time.Hour

// The expiry is signed and verified on the server. Binding the signature to the
// current credential hash also invalidates cookies after password changes or
// account removal/recreation without placing the hash in the cookie itself.
func (a *App) setSessionCookie(w http.ResponseWriter, r *http.Request, id, credentialHash string) {
	expires := time.Now().Add(sessionLifetime)
	payload := id + "|" + strconv.FormatInt(expires.Unix(), 10)
	signature := a.sign("session|" + payload + "|" + credentialHash)
	http.SetCookie(w, &http.Cookie{
		Name: "session", Value: base64.StdEncoding.EncodeToString([]byte(payload + "|" + signature)),
		Path: "/", HttpOnly: true, Secure: a.requestScheme(r) == "https", SameSite: http.SameSiteLaxMode, Expires: expires,
	})
}

func (a *App) getSessionID(r *http.Request) (string, bool) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return "", false
	}
	raw, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		return "", false
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 || !ruleset.AllowedUserID(parts[0]) {
		return "", false
	}
	expires, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() >= expires {
		return "", false
	}
	users, err := user.LoadUsers(a.authListFile)
	if err != nil {
		return "", false
	}
	hash, exists := users[parts[0]]
	if !exists {
		return "", false
	}
	expected := a.sign("session|" + parts[0] + "|" + parts[1] + "|" + hash)
	if !hmac.Equal([]byte(parts[2]), []byte(expected)) {
		return "", false
	}
	return parts[0], true
}

func (a *App) sign(value string) string {
	mac := hmac.New(sha256.New, a.sessionKey)
	mac.Write([]byte(value))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// Only an explicitly trusted immediate proxy may describe the external scheme.
// Multiple/comma-separated values are ambiguous and are not accepted.
func (a *App) requestScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	values := r.Header.Values("X-Forwarded-Proto")
	if a.isTrustedProxy(host) && len(values) == 1 && strings.EqualFold(strings.TrimSpace(values[0]), "https") {
		return "https"
	}
	return "http"
}
