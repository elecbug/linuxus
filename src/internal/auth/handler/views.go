package handler

import (
	"bytes"
	"log"
	"net/http"

	"github.com/elecbug/linuxus/src/internal/auth/page"
)

func (a *App) pageData(title string) page.Data {
	base := "/" + a.adminPath
	return page.Data{
		Title: title, AllowSignup: a.allowSignup,
		LoginURL: "/" + a.loginPath, SignupURL: "/" + a.signupPath,
		ServiceURL: "/" + a.servicePath + "/", TerminalURL: "/" + a.terminalPath + "/",
		LogoutURL: "/" + a.logoutPath, AdminURL: base,
		BaseURL: base, APIURL: base + "/api/users", AdminID: a.adminID,
	}
}

func (a *App) applicationData(r *http.Request, id, view string) page.Data {
	title := "Workspace"
	if view == "admin" {
		title = "Administration"
	}
	data := a.pageData(title)
	data.ID, data.View = id, view
	data.IsAdmin = a.adminID != "" && id == a.adminID
	if data.IsAdmin {
		data.Token = a.adminToken(r)
	}
	return data
}

// renderPage buffers output so template failures cannot leave a partial response.
func (a *App) renderPage(w http.ResponseWriter, name string, data page.Data, status int) {
	var buf bytes.Buffer
	if err := webTemplates.ExecuteTemplate(&buf, name, data); err != nil {
		log.Printf("failed to render %s page: %v", name, err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}
