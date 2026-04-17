package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// managerUserUpRequest is the payload for manager container-prepare requests.
type managerUserUpRequest struct {
	// UserID is the original user identifier.
	UserID string `json:"user_id"`
	// SafeID is the sanitized user identifier used in runtime naming.
	SafeID string `json:"safe_id"`
}

// managerUserUpResponse is the manager response for container-prepare requests.
type managerUserUpResponse struct {
	// OK indicates whether the manager completed the prepare operation.
	OK bool `json:"ok"`
	// UserID echoes the original user identifier.
	UserID string `json:"user_id"`
	// SafeID echoes the sanitized user identifier.
	SafeID string `json:"safe_id"`
	// ContainerName is the prepared container name.
	ContainerName string `json:"container_name"`
	// NetworkName is the prepared network name.
	NetworkName string `json:"network_name"`
	// Subnet is the network subnet assigned by the manager.
	Subnet string `json:"subnet"`
	// Message carries human-readable status details.
	Message string `json:"message"`
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
