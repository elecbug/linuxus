package handler

import "net/http"

func (a *App) handleServiceRedirect(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.getSessionID(r); !ok {
		http.Redirect(w, r, "/"+a.loginPath, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/"+a.servicePath+"/", http.StatusSeeOther)
}

func (a *App) handleServicePage(w http.ResponseWriter, r *http.Request) {
	id, ok := a.getSessionID(r)
	if !ok {
		http.Redirect(w, r, "/"+a.loginPath, http.StatusSeeOther)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	data := struct {
		ID string
	}{
		ID: id,
	}

	if err := a.serviceTmpl.Execute(w, data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}
