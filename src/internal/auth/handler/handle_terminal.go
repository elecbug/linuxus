package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// handleTerminalRedirect normalizes terminal route access for authenticated users.
func (a *App) handleTerminalRedirect(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.getSessionID(r); !ok {
		http.Redirect(w, r, "/"+a.loginPath, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/"+a.terminalPath+"/", http.StatusSeeOther)
}

// handleTerminalProxy validates session and proxies terminal traffic to user runtime.
func (a *App) handleTerminalProxy(w http.ResponseWriter, r *http.Request) {
	id, ok := a.getSessionID(r)
	if !ok {
		http.Redirect(w, r, "/"+a.loginPath, http.StatusSeeOther)
		return
	}

	containerName, err := a.ensureUserContainerReady(r.Context(), id)
	if err != nil {
		log.Printf("manager prepare failed for %s: %v", id, err)
		a.renderError(w, "Shell container is not ready. Please try again later.", http.StatusServiceUnavailable)
		return
	}

	targetURL := fmt.Sprintf("http://%s:7681", containerName)

	target, err := url.Parse(targetURL)
	if err != nil {
		a.renderError(w, "Invalid backend target", http.StatusInternalServerError)
		return
	}

	proxy := a.terminalProxy(target)

	if isWebSocketRequest(r) {
		ctx, cancel := context.WithCancel(r.Context())
		r = r.WithContext(ctx)
		a.sessionMu.Lock()
		if a.sessionCancels == nil {
			a.sessionCancels = make(map[*http.Request]context.CancelFunc)
		}
		a.sessionCancels[r] = cancel
		a.sessionMu.Unlock()
		defer func() { cancel(); a.sessionMu.Lock(); delete(a.sessionCancels, r); a.sessionMu.Unlock() }()
		a.markSessionStart(id)
		defer a.markSessionEnd(id)
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error for %s: %v", id, err)
		a.renderError(w, "Shell backend is unavailable", http.StatusBadGateway)
	}

	proxy.ServeHTTP(w, r)
}

// terminalProxy strips Auth's credential before crossing into a user's runtime.
func (a *App) terminalProxy(target *url.URL) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{Rewrite: func(p *httputil.ProxyRequest) {
		p.SetURL(target)
		path := strings.TrimPrefix(p.Out.URL.Path, "/"+a.terminalPath)
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		p.Out.URL.Path = path
		p.Out.URL.RawPath = ""

		p.SetXForwarded()
		p.Out.Header.Set("X-Forwarded-For", a.clientIP(p.In))
		p.Out.Header.Set("X-Forwarded-Proto", a.requestScheme(p.In))
		cookies := p.Out.Cookies()
		p.Out.Header.Del("Cookie")
		for _, cookie := range cookies {
			if cookie.Name != "session" {
				p.Out.AddCookie(cookie)
			}
		}
	}}
}

// isWebSocketRequest matches complete Connection tokens, including repeated headers.
func isWebSocketRequest(r *http.Request) bool {
	if !strings.EqualFold(strings.TrimSpace(r.Header.Get("Upgrade")), "websocket") {
		return false
	}
	for _, value := range r.Header.Values("Connection") {
		for _, token := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(token), "upgrade") {
				return true
			}
		}
	}
	return false
}

// markSessionStart increments active session count and reports it to manager.
func (a *App) markSessionStart(id string) {
	a.sessionMu.Lock()
	a.activeSessions[id]++
	current := a.activeSessions[id]
	observedAt := time.Now()
	a.sessionMu.Unlock()

	go a.reportSessionSnapshot(id, current, observedAt)
}

// markSessionEnd decrements the active session count for a user and reports the updated state to the manager.
func (a *App) markSessionEnd(id string) {
	a.sessionMu.Lock()
	if a.activeSessions[id] > 0 {
		a.activeSessions[id]--
	}
	current := a.activeSessions[id]
	observedAt := time.Now()
	a.sessionMu.Unlock()

	go a.reportSessionSnapshot(id, current, observedAt)
}

func (a *App) reportSessionSnapshot(id string, active int, observedAt time.Time) {
	if err := a.reportSessionState(id, active, observedAt); err != nil {
		log.Printf("failed to report session state for %s: %v", id, err)
	}
}
