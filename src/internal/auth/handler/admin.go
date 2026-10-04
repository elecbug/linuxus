package handler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/elecbug/linuxus/src/internal/common/user"
)

func (a *App) requireAdmin(w http.ResponseWriter, r *http.Request, redirectToLogin bool) bool {
	w.Header().Set("Cache-Control", "no-store")
	id, ok := a.getSessionID(r)
	if !ok {
		if redirectToLogin {
			http.Redirect(w, r, "/"+a.loginPath, http.StatusSeeOther)
		} else {
			http.Error(w, "authentication required", http.StatusUnauthorized)
		}
		return false
	}
	if a.adminID == "" || id != a.adminID {
		http.Error(w, "administrator access required", 403)
		return false
	}
	return true
}

func (a *App) adminToken(r *http.Request) string {
	cookie, err := r.Cookie("session")
	if err != nil {
		return ""
	}
	return a.sign("admin-csrf|" + cookie.Value)
}

func (a *App) managerOperation(ctx context.Context, path, method string, payload any, result any) error {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, a.managerBaseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Manager-Session-Secret", a.managerSessionSecret)
	resp, err := a.managerClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("Manager operation failed (HTTP %d)", resp.StatusCode)
	}
	if result != nil {
		if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(result); err != nil {
			return err
		}
		if warning := resp.Header.Get("X-Linuxus-Disk-Error"); warning != "" {
			return fmt.Errorf("%s", warning)
		}
	}
	return nil
}

func (a *App) handleAdmin(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/"+a.adminPath && r.URL.Path != "/"+a.adminPath+"/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	if !a.requireAdmin(w, r, true) {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Frame-Options", "DENY")
	if err := adminTemplate.Execute(w, map[string]string{
		"Token": a.adminToken(r), "BaseURL": "/" + a.adminPath,
		"APIURL": "/" + a.adminPath + "/api/users", "LoginURL": "/" + a.loginPath,
		"ServiceURL": "/" + a.servicePath + "/", "LogoutURL": "/" + a.logoutPath, "AdminID": a.adminID,
	}); err != nil {
		return
	}
}

func (a *App) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r, false) {
		return
	}
	if r.Method == http.MethodGet {
		accounts, err := user.LoadUsers(a.authListFile)
		if err != nil {
			http.Error(w, "cannot read accounts", 500)
			return
		}
		var states map[string]map[string]any
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		stateErr := a.managerOperation(ctx, "/admin/status", http.MethodGet, nil, &states)
		if states == nil && stateErr == nil {
			stateErr = fmt.Errorf("Manager returned an invalid status response")
		}
		ids := make([]string, 0, len(accounts))
		for id := range accounts {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		rows := make([]map[string]any, 0, len(ids))
		for _, id := range ids {
			account, err := user.ParseAccount(accounts[id])
			if err != nil {
				http.Error(w, "invalid account record", 500)
				return
			}
			row := states[id]
			if row == nil {
				row = map[string]any{}
			}
			if row["state"] == nil {
				row["state"] = "stopped"
				if states == nil {
					row["state"] = "unknown"
				}
			}
			row["id"] = id
			row["locked"] = account.Locked || account.Maintenance
			row["maintenance"] = account.Maintenance
			row["class"] = account.Class
			row["template"] = account.Template
			rows = append(rows, row)
		}
		result := map[string]any{"users": rows, "templates": a.templates, "classes": a.classes, "csrf_token": a.adminToken(r), "status_available": states != nil}
		if stateErr != nil {
			result["warning"] = stateErr.Error()
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	if token := r.Header.Get("X-CSRF-Token"); token == "" || !hmac.Equal([]byte(token), []byte(a.adminToken(r))) {
		http.Error(w, "invalid CSRF token", 403)
		return
	}
	var request struct {
		UserID string `json:"user_id"`
		Action string `json:"action"`
		Value  string `json:"value"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	action := request.Action
	switch action {
	case "lock":
		if request.UserID == a.adminID {
			http.Error(w, "cannot lock the current administrator", 400)
			return
		}
	case "unlock", "password", "disconnect", "restart":
	case "template":
		if _, ok := a.templates[request.Value]; !ok && request.Value != "default" {
			http.Error(w, "unknown template", 400)
			return
		}
	case "class":
		if _, ok := a.classes[request.Value]; !ok {
			http.Error(w, "unknown class", 400)
			return
		}
	default:
		http.Error(w, "unsupported operation", 400)
		return
	}
	if action == "restart" {
		action = "disconnect"
	}
	if err := user.UpdateAccount(a.authListFile, request.UserID, action, request.Value); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	w.Header().Set("X-Linuxus-Account-Updated", "true")
	if request.UserID == a.adminID {
		w.Header().Set("X-Linuxus-Reauthenticate", "true")
	}
	if action != "unlock" {
		ctx, cancel := context.WithTimeout(r.Context(), 50*time.Second)
		defer cancel()
		operation := "stop"
		if request.Action == "restart" {
			operation = "restart"
		}
		if err := a.managerOperation(ctx, "/admin/user", http.MethodPost, map[string]string{"user_id": request.UserID, "action": operation}, nil); err != nil {
			http.Error(w, "Account updated; "+err.Error(), 503)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"ok":true}`)
}
