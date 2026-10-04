package handler

import (
	"bytes"
	"embed"
	"html/template"
	"net/http"
	"strings"
	"time"
)

//go:embed admin.html admin.css admin.js
var adminAssets embed.FS

var adminTemplate = template.Must(template.ParseFS(adminAssets, "admin.html"))

func (a *App) handleAdminAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !a.requireAdmin(w, r, false) {
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/"+a.adminPath+"/assets/")
	if name != "admin.css" && name != "admin.js" {
		http.NotFound(w, r)
		return
	}
	data, err := adminAssets.ReadFile(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
}
