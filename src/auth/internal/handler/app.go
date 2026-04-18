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

// App holds runtime state, templates, and handlers for the auth web service.
type App struct {
	// users maps login IDs to password hashes.
	users map[string]string
	// sessionKey is used to sign session cookie payloads.
	sessionKey []byte
	// loginTmpl renders the login page.
	loginTmpl *template.Template
	// serviceTmpl renders the service page.
	serviceTmpl *template.Template
	// errorTmpl renders generic error pages.
	errorTmpl *template.Template
	// loginPath is the login endpoint path segment.
	loginPath string
	// logoutPath is the logout endpoint path segment.
	logoutPath string
	// servicePath is the service endpoint path segment.
	servicePath string
	// terminalPath is the terminal proxy endpoint path segment.
	terminalPath string
	// userContainerNamePrefix prefixes target user container names.
	userContainerNamePrefix string
	// trustedProxies contains CIDR networks treated as trusted forwarders.
	trustedProxies []*net.IPNet

	// managerBaseURL is the manager service base URL.
	managerBaseURL string
	// managerClient performs HTTP calls to the manager service.
	managerClient *http.Client

	// mu protects login failure tracking maps.
	mu sync.Mutex
	// ipFails tracks per-IP login failure state.
	ipFails map[string]*loginAttempt
	// userFails tracks per-user login failure state.
	userFails map[string]*loginAttempt

	// done signals the background cleanup goroutine to stop.
	done chan struct{}

	// mux is the HTTP request multiplexer.
	mux *http.ServeMux

	// session aggregation
	sessionMu      sync.Mutex
	activeSessions map[string]int

	// optional: report timeout / retry tuning
	sessionReportTimeout time.Duration
}

// AppConfig defines startup configuration for the auth application.
type AppConfig struct {
	// Users maps user IDs to bcrypt password hashes.
	Users map[string]string
	// SessionKey signs session cookie payloads.
	SessionKey []byte
	// LoginPath is the login route path segment.
	LoginPath string
	// LogoutPath is the logout route path segment.
	LogoutPath string
	// ServicePath is the service page route path segment.
	ServicePath string
	// TerminalPath is the terminal proxy route path segment.
	TerminalPath string
	// UserContainerNamePrefix prefixes per-user runtime container names.
	UserContainerNamePrefix string
	// TrustedProxies lists trusted proxy CIDR strings.
	TrustedProxies []string

	// ManagerBaseURL is the base URL for manager API calls.
	ManagerBaseURL string
	// ManagerTimeout is the timeout used for manager API requests.
	ManagerTimeout time.Duration
}

// loginAttempt stores throttling and lockout state for a principal.
type loginAttempt struct {
	// FailCount is the number of recent failures within the rolling window.
	FailCount int
	// LockCount is the number of lock events already applied.
	LockCount int
	// LockedUntil is the timestamp until which login is blocked.
	LockedUntil time.Time
	// LastFailAt is the timestamp of the latest failed attempt.
	LastFailAt time.Time
}

// NewApp initializes an App with validated defaults and background cleanup.
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

	timeout := config.ManagerTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
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
		managerBaseURL:          strings.TrimRight(config.ManagerBaseURL, "/"),
		managerClient: &http.Client{
			Timeout: timeout,
		},
		mux: http.NewServeMux(),

		mu:        sync.Mutex{},
		ipFails:   make(map[string]*loginAttempt),
		userFails: make(map[string]*loginAttempt),

		done: make(chan struct{}),

		sessionMu:            sync.Mutex{},
		activeSessions:       make(map[string]int),
		sessionReportTimeout: 5 * time.Second,
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

// Muxer returns the HTTP serve mux used by the app.
func (a *App) Muxer() *http.ServeMux {
	return a.mux
}

// LoginPath returns the configured login route path segment.
func (a *App) LoginPath() string {
	return a.loginPath
}

// LogoutPath returns the configured logout route path segment.
func (a *App) LogoutPath() string {
	return a.logoutPath
}

// ServicePath returns the configured service route path segment.
func (a *App) ServicePath() string {
	return a.servicePath
}

// TerminalPath returns the configured terminal route path segment.
func (a *App) TerminalPath() string {
	return a.terminalPath
}

// Start starts the HTTP server on the provided address.
func (a *App) Start(addr string) error {
	log.Printf("Auth server listening on %s", addr)
	return http.ListenAndServe(addr, a.Muxer())
}

// Stop signals background maintenance goroutines to terminate.
func (a *App) Stop() {
	close(a.done)
}

// RegisterRoutes compiles templates and binds all HTTP handlers.
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

// evictStaleEntries removes old login failure entries that are no longer relevant.
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

// getSessionID validates and extracts a session ID from the request cookie.
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
