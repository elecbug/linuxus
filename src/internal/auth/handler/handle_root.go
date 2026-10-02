package handler

import (
	"bytes"
	"net/http"
	"time"

	"github.com/elecbug/linuxus/src/static"
)

// handleRoot redirects authenticated users to service and others to login.
func (a *App) handleRoot(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.getSessionID(r); ok {
		http.Redirect(w, r, "/"+a.servicePath+"/", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/"+a.loginPath, http.StatusSeeOther)
}

// handleFavicon serves embedded assets, independent of the working directory.
func (a *App) handleFavicon() {
	a.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(static.Files))))
	a.mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		data, _ := static.Files.ReadFile("favicon.png")
		http.ServeContent(w, r, "favicon.png", time.Time{}, bytes.NewReader(data))
	})
}
