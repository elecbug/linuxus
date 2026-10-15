package handler

import (
	"crypto/hmac"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/elecbug/linuxus/src/internal/common/user"
)

func (a *App) settingsToken(r *http.Request) string {
	cookie, err := r.Cookie("session")
	if err != nil {
		return ""
	}
	return a.sign("settings-csrf|" + cookie.Value)
}

// handleSettings always selects the account from its signed session, never input.
func (a *App) handleSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Frame-Options", "DENY")
	if r.URL.Path != "/"+a.settingsPath && r.URL.Path != "/"+a.settingsPath+"/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, record, ok := a.getSessionAccount(r)
	jsonResponse := r.Header.Get("Accept") == "application/json"
	fail := func(message string, status int) {
		if jsonResponse {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
			return
		}
		if !ok {
			http.Redirect(w, r, "/"+a.loginPath, http.StatusSeeOther)
			return
		}
		data := a.applicationData(r, id, "settings")
		data.Error = message
		a.renderPage(w, "app", data, status)
	}
	if !ok {
		fail("Your session expired. Sign in again to continue.", http.StatusUnauthorized)
		return
	}
	if r.Method == http.MethodGet {
		a.renderPage(w, "app", a.applicationData(r, id, "settings"), http.StatusOK)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		fail("Unable to read the form. Please try again.", http.StatusBadRequest)
		return
	}
	token := r.PostForm.Get("csrf_token")
	if token == "" || !hmac.Equal([]byte(token), []byte(a.settingsToken(r))) {
		fail("Invalid request token. Reload Settings before trying again.", http.StatusForbidden)
		return
	}
	current, password, confirmation := r.PostForm.Get("current_password"), r.PostForm.Get("new_password"), r.PostForm.Get("confirm_password")
	if current == "" || password == "" || confirmation == "" {
		fail("Complete all password fields.", http.StatusBadRequest)
		return
	}
	if len(current) > 72 || len(password) > 72 {
		fail("Passwords must be at most 72 UTF-8 bytes.", http.StatusBadRequest)
		return
	}
	if password != confirmation {
		fail("New passwords do not match.", http.StatusBadRequest)
		return
	}
	ip := a.clientIP(r)
	if blocked, until := a.isBlocked(ip, id); blocked {
		w.Header().Set("Retry-After", strconv.Itoa(int(time.Until(until).Seconds())+1))
		fail("Too many password attempts. Please try again later.", http.StatusTooManyRequests)
		return
	}
	err := user.ChangePassword(a.authListFile, id, record, current, password)
	if errors.Is(err, user.ErrPasswordMismatch) {
		a.recordFail(ip, id, true)
		fail("Current password is incorrect.", http.StatusBadRequest)
		return
	}
	if errors.Is(err, user.ErrAccountChanged) {
		ok = false
		a.clearSessionCookie(w, r)
		fail("Your account changed. Sign in again before changing the password.", http.StatusUnauthorized)
		return
	}
	if err != nil {
		log.Printf("password change failed for %s: %v", id, err)
		fail("Unable to save your password. Please try again later.", http.StatusInternalServerError)
		return
	}
	a.clearFail(ip, id)
	a.clearSessionCookie(w, r)
	a.cancelRevokedSessions()
	w.Header().Set("X-Linuxus-Reauthenticate", "true")
	if jsonResponse {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{"login_url": "/" + a.loginPath + "?password=changed"})
		return
	}
	http.Redirect(w, r, "/"+a.loginPath+"?password=changed", http.StatusSeeOther)
}
