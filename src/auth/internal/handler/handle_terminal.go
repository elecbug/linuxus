package handler

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// handleTerminalRedirect validates session and normalizes terminal route path.
func (a *App) handleTerminalRedirect(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.getSessionID(r); !ok {
		http.Redirect(w, r, "/"+a.loginPath, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/"+a.terminalPath+"/", http.StatusSeeOther)
}

// handleTerminalProxy ensures runtime readiness and proxies traffic to the user shell.
func (a *App) handleTerminalProxy(w http.ResponseWriter, r *http.Request) {
	id, ok := a.getSessionID(r)
	if !ok {
		http.Redirect(w, r, "/"+a.loginPath, http.StatusSeeOther)
		return
	}

	if err := a.ensureUserContainerReady(r.Context(), id); err != nil {
		log.Printf("manager prepare failed for %s: %v", id, err)
		a.renderError(w, "Shell container is not ready. Please try again later.", http.StatusServiceUnavailable)
		return
	}

	safeID := sanitizeID(id)
	targetURL := fmt.Sprintf("http://%s%s:7681", a.userContainerNamePrefix, safeID)

	target, err := url.Parse(targetURL)
	if err != nil {
		a.renderError(w, "Invalid backend target", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	isWS := isWebSocketRequest(r)
	if isWS {
		a.markSessionStart(id)
		defer a.markSessionEnd(id)
	}

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host

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
		a.renderError(w, "Shell backend is unavailable", http.StatusBadGateway)
	}

	proxy.ServeHTTP(w, r)
}

// sanitizeID converts user IDs into safe backend hostname fragments.
func sanitizeID(id string) string {
	id = strings.ToLower(id)
	var b strings.Builder

	for _, ch := range id {
		if (ch >= 'a' && ch <= 'z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '_' || ch == '-' || ch == '.' {
			b.WriteRune(ch)
		} else {
			b.WriteRune('_')
		}
	}

	result := b.String()
	result = strings.TrimLeft(result, "._-")
	if result == "" {
		return "invalid"
	}
	return result
}

// isWebSocketRequest checks if the incoming HTTP request is a WebSocket upgrade request.
func isWebSocketRequest(r *http.Request) bool {
	connection := strings.ToLower(r.Header.Get("Connection"))
	upgrade := strings.ToLower(r.Header.Get("Upgrade"))

	return strings.Contains(connection, "upgrade") && upgrade == "websocket"
}

// ensureUserContainerReady asks the manager to prepare the user runtime and waits for it to be ready.
func (a *App) markSessionStart(id string) {
	fmt.Printf("Ensuring container ready for user %s\n", id)
	a.sessionMu.Lock()
	a.activeSessions[id]++
	current := a.activeSessions[id]
	a.sessionMu.Unlock()

	if err := a.reportSessionState(id, current); err != nil {
		log.Printf("failed to report session start for %s: %v", id, err)
	}
}

// markSessionEnd decrements the active session count for a user and reports the updated state to the manager.
func (a *App) markSessionEnd(id string) {
	fmt.Printf("Session ended for user %s\n", id)
	a.sessionMu.Lock()
	if a.activeSessions[id] > 0 {
		a.activeSessions[id]--
	}
	current := a.activeSessions[id]
	a.sessionMu.Unlock()

	if err := a.reportSessionState(id, current); err != nil {
		log.Printf("failed to report session end for %s: %v", id, err)
	}
}
