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

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024))

	var parsed managerUserUpResponse
	_ = json.Unmarshal(respBody, &parsed)

	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(parsed.Message)
		if msg == "" {
			msg = strings.TrimSpace(string(respBody))
		}
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("manager rejected request: %s", msg)
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
