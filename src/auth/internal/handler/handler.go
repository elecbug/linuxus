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

const LOGIN_PATH = "login"
const LOGOUT_PATH = "logout"
const SERVICE_PATH = "service"
const TERMINAL_PATH = "terminal"

type App struct {
	users       map[string]string
	sessionKey  []byte
	loginTmpl   *template.Template
	serviceTmpl *template.Template
}

func NewApp(users map[string]string, sessionKey []byte, loginTmpl, serviceTmpl *template.Template) *App {
	return &App{
		users:       users,
		sessionKey:  sessionKey,
		loginTmpl:   loginTmpl,
		serviceTmpl: serviceTmpl,
	}
}

func (a *App) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", a.handleRoot)
	mux.HandleFunc("/"+LOGIN_PATH, a.handleLogin)
	mux.HandleFunc("/"+LOGOUT_PATH, a.handleLogout)

	mux.HandleFunc("/"+SERVICE_PATH, a.handleServiceRedirect)
	mux.HandleFunc("/"+SERVICE_PATH+"/", a.handleServicePage)
	mux.HandleFunc("/"+TERMINAL_PATH, a.handleTerminalRedirect)
	mux.HandleFunc("/"+TERMINAL_PATH+"/", a.handleTerminalProxy)
}

func (a *App) handleRoot(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.getSessionStudentID(r); ok {
		http.Redirect(w, r, "/"+SERVICE_PATH+"/", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/"+LOGIN_PATH, http.StatusSeeOther)
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

		studentID := strings.TrimSpace(r.FormValue("student_id"))
		password := r.FormValue("password")

		hash, ok := a.users[studentID]
		if !ok {
			a.renderLogin(w, "Invalid student ID or password")
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
			a.renderLogin(w, "Invalid student ID or password")
			return
		}

		a.setSessionCookie(w, studentID)
		http.Redirect(w, r, "/"+SERVICE_PATH+"/", http.StatusSeeOther)
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

	http.Redirect(w, r, "/"+LOGIN_PATH, http.StatusSeeOther)
}

func (a *App) handleServiceRedirect(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.getSessionStudentID(r); !ok {
		http.Redirect(w, r, "/"+LOGIN_PATH, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/"+SERVICE_PATH+"/", http.StatusSeeOther)
}

func (a *App) handleServicePage(w http.ResponseWriter, r *http.Request) {
	studentID, ok := a.getSessionStudentID(r)
	if !ok {
		http.Redirect(w, r, "/"+LOGIN_PATH, http.StatusSeeOther)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	data := struct {
		StudentID string
	}{
		StudentID: studentID,
	}

	if err := a.serviceTmpl.Execute(w, data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (a *App) handleTerminalRedirect(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.getSessionStudentID(r); !ok {
		http.Redirect(w, r, "/"+LOGIN_PATH, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/"+TERMINAL_PATH+"/", http.StatusSeeOther)
}

func (a *App) handleTerminalProxy(w http.ResponseWriter, r *http.Request) {
	studentID, ok := a.getSessionStudentID(r)
	if !ok {
		http.Redirect(w, r, "/"+LOGIN_PATH, http.StatusSeeOther)
		return
	}

	safeID := sanitizeStudentID(studentID)
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
		newPath := strings.TrimPrefix(req.URL.Path, "/"+TERMINAL_PATH)
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
		log.Printf("proxy error for %s: %v", studentID, err)
		http.Error(w, "Shell backend is unavailable", http.StatusBadGateway)
	}

	proxy.ServeHTTP(w, r)
}
