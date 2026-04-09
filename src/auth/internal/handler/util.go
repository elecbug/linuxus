package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net"
	"net/http"
	"strings"
	"time"
)

var dummyHash = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")

func (a *App) renderLogin(w http.ResponseWriter, errMsg string) {
	data := struct {
		Error string
	}{
		Error: errMsg,
	}

	if err := a.loginTmpl.Execute(w, data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (a *App) setSessionCookie(w http.ResponseWriter, id string) {
	signature := a.sign(id)
	payload := id + "|" + signature
	value := base64.StdEncoding.EncodeToString([]byte(payload))

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(12 * time.Hour),
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

	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return "", false
	}

	id := parts[0]
	signature := parts[1]

	expected := a.sign(id)
	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return "", false
	}

	return id, true
}

func (a *App) sign(value string) string {
	mac := hmac.New(sha256.New, a.sessionKey)
	mac.Write([]byte(value))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func sanitizeID(id string) string {
	id = strings.ToLower(id)
	var b strings.Builder

	for _, ch := range id {
		if (ch >= 'a' && ch <= 'z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '_' || ch == '-' || ch == '.' {
			b.WriteRune(ch)
		} else {
			b.WriteRune('_')
		}
	}

	result := b.String()
	result = strings.TrimLeft(result, "._-")
	if result == "" {
		return "invalid"
	}
	return result
}

func (a *App) isBlocked(ip, id string) (bool, time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()

	if s, ok := a.ipFails[ip]; ok && now.Before(s.LockedUntil) {
		return true, s.LockedUntil
	}
	if s, ok := a.userFails[id]; ok && now.Before(s.LockedUntil) {
		return true, s.LockedUntil
	}
	return false, time.Time{}
}

func (a *App) recordFail(ip, id string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()

	update := func(m map[string]*LoginAttempt, key string, limit int) {
		s, ok := m[key]
		if !ok {
			s = &LoginAttempt{}
			m[key] = s
		}

		if now.Sub(s.LastFailAt) > 15*time.Minute {
			s.FailCount = 0
		}

		if !s.LockedUntil.IsZero() && now.After(s.LockedUntil) && now.Sub(s.LockedUntil) > 30*time.Minute {
			s.LockCount = 0
		}

		s.FailCount++
		s.LastFailAt = now

		if s.FailCount >= limit {
			s.LockCount++
			s.LockedUntil = now.Add(lockDuration(s.LockCount))
			s.FailCount = 0
		}
	}

	update(a.ipFails, ip, 20)
	update(a.userFails, id, 5)
}

func (a *App) clearFail(ip, id string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.ipFails, ip)
	delete(a.userFails, id)
}

func (a *App) failDelay(ip, id string) time.Duration {
	a.mu.Lock()
	defer a.mu.Unlock()

	n := 1
	if s, ok := a.userFails[id]; ok && s.FailCount > n {
		n = s.FailCount
	}
	if n > 5 {
		n = 5
	}

	return time.Duration(n) * 300 * time.Millisecond
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func printTime(d time.Time) string {
	if time.Until(d) < 24*time.Hour {
		return d.Format("15:04:05")
	}
	return d.Format("2006.01.02. 15:04:05")
}

func lockDuration(lockCount int) time.Duration {
	switch lockCount {
	case 1:
		return 1 * time.Minute
	case 2:
		return 2 * time.Minute
	case 3:
		return 4 * time.Minute
	default:
		return 8 * time.Minute
	}
}
