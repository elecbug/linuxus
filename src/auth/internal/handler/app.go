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

	"github.com/elecbug/linuxus/src/auth/internal/page"
)

// App holds auth server state, templates, and manager integration clients.
type App struct {
	// users maps user IDs to bcrypt password hashes.
	users map[string]string
	// sessionKey is used to sign session cookie payloads.
	sessionKey []byte
	// loginTmpl renders the login page.
	loginTmpl *template.Template
	// serviceTmpl renders the service landing page.
	serviceTmpl *template.Template
	// errorTmpl renders error pages.
	errorTmpl *template.Template
	// loginPath is the configured login endpoint path.
	loginPath string
	// logoutPath is the configured logout endpoint path.
	logoutPath string
	// servicePath is the configured service endpoint path.
	servicePath string
	// terminalPath is the configured terminal endpoint path.
	terminalPath string
	// userContainerNamePrefix is prefixed to user runtime container names.
	userContainerNamePrefix string
	// trustedProxies contains CIDR networks allowed to forward client IP headers.
	trustedProxies []*net.IPNet

	// managerBaseURL is the manager service base URL.
	managerBaseURL string
	// managerClient is the HTTP client used to call manager endpoints.
	managerClient *http.Client
	// managerSecret is an optional secret used for manager-authenticated requests.
	managerSecret string

	// mu protects login failure tracking maps.
	mu sync.Mutex
	// ipFails tracks failed attempts by source IP.
	ipFails map[string]*loginAttempt
	// userFails tracks failed attempts by user ID.
	userFails map[string]*loginAttempt

	// done signals background goroutines to stop.
	done chan struct{}

	// mux is the HTTP request multiplexer.
	mux *http.ServeMux

	// sessionMu protects active session counters.
	sessionMu sync.Mutex
	// activeSessions tracks active terminal sessions per user.
	activeSessions map[string]int

	// sessionReportTimeout limits manager reporting request duration.
	sessionReportTimeout time.Duration

	// AllowSignup indicates whether new user signups are allowed.
	allowSignup bool
}

// AppConfig defines all configuration values required to initialize an App.
type AppConfig struct {
	// Users maps user IDs to bcrypt password hashes.
	Users map[string]string
	// SessionKey is used to sign session cookie payloads.
	SessionKey []byte
	// LoginPath is the route path for login.
	LoginPath string
	// LogoutPath is the route path for logout.
	LogoutPath string
	// ServicePath is the route path for service page.
	ServicePath string
	// TerminalPath is the route path for terminal proxy.
	TerminalPath string
	// UserContainerNamePrefix is prefixed to user runtime container names.
	UserContainerNamePrefix string
	// TrustedProxies contains CIDR ranges trusted as reverse proxies.
	TrustedProxies []string

	// ManagerBaseURL is the manager service base URL.
	ManagerBaseURL string
	// ManagerTimeout is the HTTP timeout for manager requests.
	ManagerTimeout time.Duration
	// ManagerSecret is an optional shared secret for manager requests.
	ManagerSecret string
	// AllowSignup indicates whether new user signups are allowed.
	AllowSignup bool
}

// loginAttempt stores rolling failure counters and lock metadata.
type loginAttempt struct {
	// FailCount is the number of recent failed attempts in the current window.
	FailCount int
	// LockCount is the number of times this key has been locked.
	LockCount int
	// LockedUntil is the time until which login attempts are blocked.
	LockedUntil time.Time
	// LastFailAt is the timestamp of the most recent failure.
	LastFailAt time.Time
}

// NewApp creates an App from configuration and starts background cleanup tasks.
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
		managerSecret: config.ManagerSecret,
		mux:           http.NewServeMux(),

		mu:        sync.Mutex{},
		ipFails:   make(map[string]*loginAttempt),
		userFails: make(map[string]*loginAttempt),

		done: make(chan struct{}),

		sessionMu:            sync.Mutex{},
		activeSessions:       make(map[string]int),
		sessionReportTimeout: 5 * time.Second,
		allowSignup:          config.AllowSignup,
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

// LoginPath returns the configured login path.
func (a *App) LoginPath() string {
	return a.loginPath
}

// LogoutPath returns the configured logout path.
func (a *App) LogoutPath() string {
	return a.logoutPath
}

// ServicePath returns the configured service path.
func (a *App) ServicePath() string {
	return a.servicePath
}

// TerminalPath returns the configured terminal path.
func (a *App) TerminalPath() string {
	return a.terminalPath
}

// Start launches the HTTP server using the configured route multiplexer.
func (a *App) Start(addr string) error {
	log.Printf("Auth server listening on %s", addr)
	return http.ListenAndServe(addr, a.mux)
}

// Stop signals background maintenance routines to terminate.
func (a *App) Stop() {
	close(a.done)
}

// RegisterRoutes compiles templates and binds HTTP handlers.
func (a *App) RegisterRoutes() {
	loginTmpl, err := template.New(a.loginPath).Parse(page.GetLoginPage(a.loginPath, a.allowSignup))
	if err != nil {
		log.Fatalf("failed to parse login template: %v", err)
	}

	serviceTmpl, err := template.New(a.servicePath).Parse(page.GetServicePage(a.terminalPath, a.logoutPath))
	if err != nil {
		log.Fatalf("failed to parse service template: %v", err)
	}

	errorTmpl, err := template.New("error").Parse(page.GetErrorPage())
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

// evictStaleEntries removes old failure tracking entries that are no longer active.
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

// getSessionID validates the signed session cookie and returns the user ID.
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
