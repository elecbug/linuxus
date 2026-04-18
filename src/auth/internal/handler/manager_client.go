package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type managerUserUpRequest struct {
	UserID string `json:"user_id"`
	SafeID string `json:"safe_id"`
}

type managerUserUpResponse struct {
	OK            bool   `json:"ok"`
	UserID        string `json:"user_id"`
	SafeID        string `json:"safe_id"`
	ContainerName string `json:"container_name"`
	NetworkName   string `json:"network_name"`
	Subnet        string `json:"subnet"`
	Message       string `json:"message"`
}

// sessionStateReport defines the structure of session state reports sent to the manager.
type sessionStateReport struct {
	// UserID is the original user identifier.
	UserID string `json:"user_id"`
	// ActiveSessions is the count of active sessions for the user.
	ActiveSessions int `json:"active_sessions"`
	// ObservedAt is the timestamp when the session state was observed.
	ObservedAt time.Time `json:"observed_at"`
}

// ensureUserContainerReady asks the manager service to ensure a user runtime is ready.
func (a *App) ensureUserContainerReady(ctx context.Context, userID string) error {
	if a.managerBaseURL == "" {
		return fmt.Errorf("manager base url is not configured")
	}

	payload := managerUserUpRequest{
		UserID: userID,
		SafeID: sanitizeID(userID),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal manager request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		a.managerBaseURL+"/user/up",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("failed to build manager request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.managerClient.Do(req)
	if err != nil {
		return fmt.Errorf("manager request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
	if err != nil {
		return fmt.Errorf("failed to read manager response: %w", err)
	}

	var parsed managerUserUpResponse
	unmarshalErr := json.Unmarshal(respBody, &parsed)

	if resp.StatusCode != http.StatusOK {
		if unmarshalErr == nil {
			msg := strings.TrimSpace(parsed.Message)
			if msg != "" {
				return fmt.Errorf("manager rejected request: %s", msg)
			}
		}
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("manager rejected request: %s", msg)
	}

	if unmarshalErr != nil {
		return fmt.Errorf("failed to parse manager response: %w", unmarshalErr)
	}

	if !parsed.OK {
		msg := strings.TrimSpace(parsed.Message)
		if msg == "" {
			msg = "manager returned not-ready response"
		}
		return fmt.Errorf("%s", msg)
	}

	return nil
}

// reportSessionState sends the current active session count for a user to the manager.
func (a *App) reportSessionState(id string, active int) error {
	if a.managerBaseURL == "" {
		return nil
	}

	payload := sessionStateReport{
		UserID:         id,
		ActiveSessions: active,
		ObservedAt:     time.Now(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal session state: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), a.sessionReportTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.managerBaseURL+"/user/session-state", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if a.managerSecret != "" {
		req.Header.Set("X-Manager-Secret", a.managerSecret)
	}

	resp, err := a.managerClient.Do(req)
	if err != nil {
		return fmt.Errorf("post session state: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("manager returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return nil
}
