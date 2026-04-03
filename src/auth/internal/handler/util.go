package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
	"time"
)

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
