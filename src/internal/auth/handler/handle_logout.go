package handler

import (
	"net/http"
)

// handleLogout removes this browser's cookie. Other open terminals keep their
// session counts until their WebSocket connections actually close.
func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   a.requestScheme(r) == "https",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	http.Redirect(w, r, "/"+a.loginPath, http.StatusSeeOther)
}
