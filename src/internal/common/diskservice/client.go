// Package diskservice connects the manager to the host's private disk service.
package diskservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/elecbug/linuxus/src/internal/common/packet"
)

const ContainerDir = "/run/linuxus-disks"
const ContainerSocket = ContainerDir + "/disk.sock"

// NewClient only dials a local Unix socket and ignores proxy settings.
func NewClient(socket string, timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		},
	}}
}

func Ensure(ctx context.Context, client *http.Client, userID string) error {
	if client == nil {
		return fmt.Errorf("disk service is not configured")
	}
	body, err := json.Marshal(packet.UserUpRequest{UserID: userID})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://disk-service/ensure-disk", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("contact host disk service: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("host disk service returned %d: %s", resp.StatusCode, strings.TrimSpace(string(message)))
	}
	return nil
}
