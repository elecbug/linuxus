package handler

import (
	"log"
	"net/http"
)

// handleLogout clears the local session state, reports logout to manager,
// removes the session cookie, and redirects to login.
func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	id, ok := a.getSessionID(r)
	if ok {
		a.sessionMu.Lock()
		delete(a.activeSessions, id)
		a.sessionMu.Unlock()

		if err := a.reportSessionState(id, 0); err != nil {
			log.Printf("failed to report logout session state for %s: %v", id, err)
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	http.Redirect(w, r, "/"+a.loginPath, http.StatusSeeOther)
}
