package handler

import "net/http"

// handleLogout clears the session cookie and redirects to login.
func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	http.Redirect(w, r, "/"+a.loginPath, http.StatusSeeOther)
}
