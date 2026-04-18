package handler

import "net/http"

func (a *App) handleRoot(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.getSessionID(r); ok {
		http.Redirect(w, r, "/"+a.servicePath+"/", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/"+a.loginPath, http.StatusSeeOther)
}
