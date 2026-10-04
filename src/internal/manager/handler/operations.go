package handler

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	commonconfig "github.com/elecbug/linuxus/src/internal/common/config"
	"github.com/elecbug/linuxus/src/internal/common/http_helper"
	"github.com/elecbug/linuxus/src/internal/common/ruleset"
	"github.com/elecbug/linuxus/src/internal/common/user"
)

func (s *Server) account(id string) (user.Account, error) {
	if s.cfg.AuthListFile == "" {
		return user.Account{}, nil
	} // Unit/in-process configurations.
	users, err := user.LoadUsers(s.cfg.AuthListFile)
	if err != nil {
		return user.Account{}, fmt.Errorf("cannot read account database: %w", err)
	}
	value, ok := users[id]
	if !ok {
		return user.Account{}, fmt.Errorf("account does not exist")
	}
	account, err := user.ParseAccount(value)
	if err != nil || account.Locked {
		return user.Account{}, fmt.Errorf("account is locked or invalid")
	}
	return account, nil
}

func (s *Server) userTemplate(id string) (string, commonconfig.Template, error) {
	account, err := s.account(id)
	if err != nil {
		return "", commonconfig.Template{}, err
	}
	name := account.Template
	if name == "" && account.Class != "" {
		var ok bool
		name, ok = s.cfg.Classes[account.Class]
		if !ok {
			return "", commonconfig.Template{}, fmt.Errorf("unknown class %s", account.Class)
		}
	}
	if name == "" || name == "default" {
		return "default", commonconfig.Template{Image: s.cfg.UserImage}, nil
	}
	template, ok := s.cfg.Templates[name]
	if !ok {
		return "", commonconfig.Template{}, fmt.Errorf("unknown environment template %s", name)
	}
	return name, template, nil
}

func (s *Server) checkAdmission(ctx context.Context) error {
	if s.cfg.MaxRunning > 0 {
		containers, err := s.docker.ContainerList(ctx, container.ListOptions{})
		if err != nil {
			return err
		}
		running := 0
		for _, info := range containers {
			for _, name := range info.Names {
				if strings.HasPrefix(strings.TrimPrefix(name, "/"), s.cfg.UserContainerNamePrefix) {
					running++
					break
				}
			}
		}
		if running >= s.cfg.MaxRunning {
			return fmt.Errorf("maximum running environments reached (%d); disconnect an idle environment or increase CAPACITY_MAX_RUNNING", s.cfg.MaxRunning)
		}
	}
	if s.cfg.MinFreeBytes > 0 {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://disk-service/capacity", nil)
		resp, err := s.diskClient.Do(req)
		if err != nil {
			return fmt.Errorf("check host storage: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("host storage reserve is unavailable; inspect doctor and CAPACITY_MIN_FREE_SPACE")
		}
	}
	return nil
}

func (s *Server) authorizeAdmin(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		http.Error(w, "method not allowed", 405)
		return false
	}
	if s.cfg.ManagerSessionSecret == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Manager-Session-Secret")), []byte(s.cfg.ManagerSessionSecret)) != 1 {
		http.Error(w, "unauthorized", 401)
		return false
	}
	return true
}

func (s *Server) HandleAdminUser(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeAdmin(w, r, http.MethodPost) {
		return
	}
	var request struct {
		UserID string `json:"user_id"`
		Action string `json:"action"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&request); err != nil || !ruleset.AllowedUserID(request.UserID) || (request.Action != "stop" && request.Action != "restart") {
		http.Error(w, "invalid operation", 400)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	if err := s.prepareMu.LockContext(ctx); err != nil {
		http.Error(w, err.Error(), 503)
		return
	}
	err := s.stopAndRemoveUserContainerAndNetwork(ctx, request.UserID)
	if err == nil {
		s.mu.Lock()
		delete(s.runtimes, request.UserID)
		s.mu.Unlock()
	}
	s.prepareMu.Unlock()
	if err == nil && request.Action == "restart" {
		_, err = s.ensureUserRuntimeReady(ctx, request.UserID)
	}
	if err != nil {
		http.Error(w, err.Error(), 503)
		return
	}
	http_helper.WriteJSONViaHTTP(w, 200, map[string]bool{"ok": true})
}

func (s *Server) HandleAdminStatus(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeAdmin(w, r, http.MethodGet) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	containers, err := s.docker.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		http.Error(w, err.Error(), 503)
		return
	}
	rows := make(map[string]map[string]any)
	s.mu.Lock()
	for id, state := range s.runtimes {
		if state != nil {
			rows[id] = map[string]any{"sessions": state.ActiveSessions, "last_observed": state.LastObservedAt}
		}
	}
	s.mu.Unlock()
	for _, info := range containers {
		for _, name := range info.Names {
			name = strings.TrimPrefix(name, "/")
			if strings.HasPrefix(name, s.cfg.UserContainerNamePrefix) {
				id := strings.TrimPrefix(name, s.cfg.UserContainerNamePrefix)
				if !ruleset.AllowedUserID(id) {
					continue
				}
				if rows[id] == nil {
					rows[id] = map[string]any{"sessions": 0}
				}
				rows[id]["state"] = info.State
				rows[id]["image"] = info.Image
			}
		}
	}
	http_helper.WriteJSONViaHTTP(w, 200, rows)
}

// Reconcile discovers runtimes after Manager restarts and gives Auth a grace
// period to replay its complete session snapshot before idle cleanup is enabled.
func (s *Server) reconcileRuntimes(ctx context.Context) error {
	containers, err := s.docker.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runtimes == nil {
		s.runtimes = make(map[string]*RuntimeState)
	}
	for _, info := range containers {
		for _, name := range info.Names {
			name = strings.TrimPrefix(name, "/")
			if !strings.HasPrefix(name, s.cfg.UserContainerNamePrefix) {
				continue
			}
			id := strings.TrimPrefix(name, s.cfg.UserContainerNamePrefix)
			if !ruleset.AllowedUserID(id) {
				continue
			}
			if s.runtimes[id] == nil {
				s.runtimes[id] = &RuntimeState{UserID: id, IdleSince: time.Now().Add(90 * time.Second)}
			}
		}
	}
	return nil
}
