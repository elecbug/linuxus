package handler

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type App struct {
	users              map[string]string
	sessionKey         []byte
	loginTmpl          *template.Template
	serviceTmpl        *template.Template
	loginPath          string
	logoutPath         string
	servicePath        string
	terminalPath       string
	adminContainerName string
}

func NewApp(
	users map[string]string,
	sessionKey []byte,
	loginPath,
	logoutPath,
	servicePath,
	terminalPath,
	adminContainerName string,
) *App {
	return &App{
		users:              users,
		sessionKey:         sessionKey,
		loginPath:          loginPath,
		logoutPath:         logoutPath,
		servicePath:        servicePath,
		terminalPath:       terminalPath,
		adminContainerName: adminContainerName,
	}
}

func (a *App) RegisterRoutes(mux *http.ServeMux) {
	loginTmpl, err := template.New(a.loginPath).Parse(a.GetLoginPage())
	if err != nil {
		log.Fatalf("failed to parse template: %v", err)
	}

	serviceTmpl, err := template.New(a.servicePath).Parse(a.GetServicePage())
	if err != nil {
		log.Fatalf("failed to parse service template: %v", err)
	}

	a.loginTmpl = loginTmpl
	a.serviceTmpl = serviceTmpl

	mux.HandleFunc("/", a.handleRoot)
	mux.HandleFunc("/"+a.loginPath, a.handleLogin)
	mux.HandleFunc("/"+a.logoutPath, a.handleLogout)

	mux.HandleFunc("/"+a.servicePath, a.handleServiceRedirect)
	mux.HandleFunc("/"+a.servicePath+"/", a.handleServicePage)
	mux.HandleFunc("/"+a.terminalPath, a.handleTerminalRedirect)
	mux.HandleFunc("/"+a.terminalPath+"/", a.handleTerminalProxy)
}

func (a *App) handleRoot(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.getSessionID(r); ok {
		http.Redirect(w, r, "/"+a.servicePath+"/", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/"+a.loginPath, http.StatusSeeOther)
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.renderLogin(w, "")
		return

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		id := strings.TrimSpace(r.FormValue("id"))
		password := r.FormValue("password")

		hash, ok := a.users[id]
		if !ok {
			a.renderLogin(w, "Invalid ID or password")
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
			a.renderLogin(w, "Invalid ID or password")
			return
		}

		a.setSessionCookie(w, id)
		http.Redirect(w, r, "/"+a.servicePath+"/", http.StatusSeeOther)
		return

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

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

func (a *App) handleTerminalRedirect(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.getSessionID(r); !ok {
		http.Redirect(w, r, "/"+a.loginPath, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/"+a.terminalPath+"/", http.StatusSeeOther)
}

func (a *App) handleTerminalProxy(w http.ResponseWriter, r *http.Request) {
	id, ok := a.getSessionID(r)
	if !ok {
		http.Redirect(w, r, "/"+a.loginPath, http.StatusSeeOther)
		return
	}

	safeID := sanitizeID(id)
	targetURL := fmt.Sprintf("http://linuxus_service_%s:7681", safeID)

	target, err := url.Parse(targetURL)
	if err != nil {
		http.Error(w, "Invalid backend target", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host

		// Strip "/$TERMINAL_PATH" prefix
		newPath := strings.TrimPrefix(req.URL.Path, "/"+a.terminalPath)
		if newPath == "" {
			newPath = "/"
		}
		if !strings.HasPrefix(newPath, "/") {
			newPath = "/" + newPath
		}
		req.URL.Path = newPath
		req.URL.RawPath = ""

		req.Header.Set("X-Forwarded-Host", r.Host)
		req.Header.Set("X-Forwarded-Proto", "http")
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error for %s: %v", id, err)
		http.Error(w, "Shell backend is unavailable", http.StatusBadGateway)
	}

	proxy.ServeHTTP(w, r)
}
