package handler

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// handleRoot redirects authenticated users to service and others to login.
func (a *App) handleRoot(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.getSessionID(r); ok {
		http.Redirect(w, r, "/"+a.servicePath+"/", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/"+a.loginPath, http.StatusSeeOther)
}

// handleFavicon serves a blank 1x1 pixel favicon to prevent 404 errors in logs.
func (a *App) handleFavicon() {
	staticDir := "static"
	if exe, err := os.Executable(); err == nil {
		staticDir = filepath.Join(filepath.Dir(exe), "static")
	} else {
		log.Printf("warning: could not determine executable path, serving static from relative path: %v", err)
	}

	a.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
	a.mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(staticDir, "favicon.png"))
	})
}
