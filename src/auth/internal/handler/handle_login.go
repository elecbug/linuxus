package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var dummyHash = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.renderLogin(w, "")
		return

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			a.renderError(w, "Bad request", http.StatusBadRequest)
			return
		}

		id := strings.TrimSpace(r.FormValue("id"))
		password := r.FormValue("password")
		ip := a.clientIP(r)

		// 1) Block check
		if ok, until := a.isBlocked(ip, id); ok {
			w.Header().Set("Retry-After", strconv.Itoa(int(time.Until(until).Seconds())+1))
			a.renderError(w, "Too many login attempts. Please try again later: "+printTime(until), http.StatusTooManyRequests)
			return
		}

		// 2) Look up user
		hash, ok := a.users[id]
		if !ok {
			hash = string(dummyHash)
		}

		// 3) Compare password
		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

		// 4) Handle failed comparison
		if err != nil {
			a.recordFail(ip, id, ok)
			a.renderLogin(w, "Invalid ID or password")
			return
		}

		// 5) Check if user does not exist
		if !ok {
			a.recordFail(ip, id, false)
			a.renderLogin(w, "Invalid ID or password")
			return
		}

		// 6) Success
		a.clearFail(ip, id)
		a.setSessionCookie(w, id)
		http.Redirect(w, r, "/"+a.servicePath+"/", http.StatusSeeOther)
		return

	default:
		a.renderError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
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

func (a *App) renderError(w http.ResponseWriter, errMsg string, statusCode int) {
	data := struct {
		Error string
	}{
		Error: errMsg,
	}

	var buf bytes.Buffer
	if err := a.errorTmpl.Execute(&buf, data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(statusCode)
	_, _ = buf.WriteTo(w)
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

func (a *App) sign(value string) string {
	mac := hmac.New(sha256.New, a.sessionKey)
	mac.Write([]byte(value))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (a *App) recordFail(ip, id string, trackUser bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()

	update := func(m map[string]*loginAttempt, key string, limit int) {
		s, ok := m[key]
		if !ok {
			s = &loginAttempt{}
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
	if trackUser {
		update(a.userFails, id, 5)
	}
}

func (a *App) clearFail(ip, id string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.ipFails, ip)
	delete(a.userFails, id)
}

func (a *App) clientIP(r *http.Request) string {
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteHost = r.RemoteAddr
	}

	if a.isTrustedProxy(remoteHost) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[0])
		}

		if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
			return strings.TrimSpace(xrip)
		}
	}

	return remoteHost
}

func (a *App) isTrustedProxy(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, network := range a.trustedProxies {
		if network.Contains(ip) {
			return true
		}
	}
	return false
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
