package handler

import (
	"crypto/hmac"
	"encoding/base64"
	"html/template"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type App struct {
	users                   map[string]string
	sessionKey              []byte
	loginTmpl               *template.Template
	serviceTmpl             *template.Template
	errorTmpl               *template.Template
	loginPath               string
	logoutPath              string
	servicePath             string
	terminalPath            string
	userContainerNamePrefix string
	trustedProxies          []*net.IPNet

	mu        sync.Mutex
	ipFails   map[string]*loginAttempt
	userFails map[string]*loginAttempt

	done chan struct{}

	mux *http.ServeMux
}

type AppConfig struct {
	Users                   map[string]string
	SessionKey              []byte
	LoginPath               string
	LogoutPath              string
	ServicePath             string
	TerminalPath            string
	UserContainerNamePrefix string
	TrustedProxies          []string
}

type loginAttempt struct {
	FailCount   int
	LockCount   int
	LockedUntil time.Time
	LastFailAt  time.Time
}

func NewApp(config *AppConfig) *App {
	var trustedProxies []*net.IPNet

	for _, cidr := range config.TrustedProxies {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil {
			trustedProxies = append(trustedProxies, network)
		} else {
			log.Printf("warning: ignoring invalid trusted proxy CIDR %q: %v", cidr, err)
		}
	}

	app := &App{
		users:                   config.Users,
		sessionKey:              config.SessionKey,
		loginPath:               config.LoginPath,
		logoutPath:              config.LogoutPath,
		servicePath:             config.ServicePath,
		terminalPath:            config.TerminalPath,
		userContainerNamePrefix: config.UserContainerNamePrefix,
		trustedProxies:          trustedProxies,
		mux:                     http.NewServeMux(),

		mu:        sync.Mutex{},
		ipFails:   make(map[string]*loginAttempt),
		userFails: make(map[string]*loginAttempt),

		done: make(chan struct{}),
	}

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				app.evictStaleEntries()
			case <-app.done:
				return
			}
		}
	}()

	return app
}

func (a *App) Muxer() *http.ServeMux {
	return a.mux
}

func (a *App) LoginPath() string {
	return a.loginPath
}

func (a *App) LogoutPath() string {
	return a.logoutPath
}

func (a *App) ServicePath() string {
	return a.servicePath
}

func (a *App) TerminalPath() string {
	return a.terminalPath
}

func (a *App) Start(addr string) error {
	log.Printf("Auth server listening on %s", addr)
	return http.ListenAndServe(addr, a.Muxer())
}

func (a *App) Stop() {
	close(a.done)
}

func (a *App) RegisterRoutes() {
	loginTmpl, err := template.New(a.loginPath).Parse(getLoginPage(a))
	if err != nil {
		log.Fatalf("failed to parse login template: %v", err)
	}

	serviceTmpl, err := template.New(a.servicePath).Parse(getServicePage(a))
	if err != nil {
		log.Fatalf("failed to parse service template: %v", err)
	}

	errorTmpl, err := template.New("error").Parse(getErrorPage())
	if err != nil {
		log.Fatalf("failed to parse error template: %v", err)
	}

	a.loginTmpl = loginTmpl
	a.serviceTmpl = serviceTmpl
	a.errorTmpl = errorTmpl

	a.mux.HandleFunc("/", a.handleRoot)
	a.mux.HandleFunc("/"+a.loginPath, a.handleLogin)
	a.mux.HandleFunc("/"+a.logoutPath, a.handleLogout)

	a.mux.HandleFunc("/"+a.servicePath, a.handleServiceRedirect)
	a.mux.HandleFunc("/"+a.servicePath+"/", a.handleServicePage)
	a.mux.HandleFunc("/"+a.terminalPath, a.handleTerminalRedirect)
	a.mux.HandleFunc("/"+a.terminalPath+"/", a.handleTerminalProxy)
}

func (a *App) evictStaleEntries() {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	const staleDuration = 30 * time.Minute

	for ip, s := range a.ipFails {
		if now.Sub(s.LastFailAt) > staleDuration && (s.LockedUntil.IsZero() || now.After(s.LockedUntil)) {
			delete(a.ipFails, ip)
		}
	}
	for id, s := range a.userFails {
		if now.Sub(s.LastFailAt) > staleDuration && (s.LockedUntil.IsZero() || now.After(s.LockedUntil)) {
			delete(a.userFails, id)
		}
	}
}

func (a *App) getSessionID(r *http.Request) (string, bool) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return "", false
	}

	raw, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		return "", false
	}

	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return "", false
	}

	id := parts[0]
	signature := parts[1]

	expected := a.sign(id)
	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return "", false
	}

	return id, true
}
