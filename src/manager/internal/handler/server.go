package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/docker/docker/client"
	"github.com/elecbug/linuxus/src/manager/internal/config"
)

var reInvalid = regexp.MustCompile(`[^a-z0-9]+`)

type Server struct {
	docker *client.Client
	cfg    *config.Config
}

func NewServer(cfg *config.Config) (*Server, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker client init failed: %w", err)
	}

	return &Server{docker: cli, cfg: cfg}, nil
}

func (s *Server) Close() error {
	if s == nil || s.docker == nil {
		return nil
	}

	return s.docker.Close()
}

func (s *Server) HandleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true,
	})
}

func (s *Server) HandleUserUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, UserUpResponse{
			OK:      false,
			Message: "method not allowed",
		})
		return
	}

	var req UserUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, UserUpResponse{
			OK:      false,
			Message: "invalid json body",
		})
		return
	}

	req.UserID = strings.TrimSpace(req.UserID)
	req.SafeID = strings.TrimSpace(req.SafeID)
	if req.UserID == "" {
		writeJSON(w, http.StatusBadRequest, UserUpResponse{
			OK:      false,
			Message: "user_id is required",
		})
		return
	}
	if req.SafeID == "" {
		req.SafeID = sanitizeName(req.UserID)
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.ManagerWaitTime)
	defer cancel()

	resp, err := s.ensureUserRuntimeReady(ctx, req.UserID, req.SafeID)
	if err != nil {
		log.Printf("user up failed user=%s safe=%s err=%v", req.UserID, req.SafeID, err)
		writeJSON(w, http.StatusServiceUnavailable, UserUpResponse{
			OK:            false,
			UserID:        req.UserID,
			SafeID:        req.SafeID,
			ContainerName: s.cfg.UserContainerNamePrefix + req.SafeID,
			Message:       err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
