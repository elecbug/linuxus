package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/elecbug/linuxus/src/internal/common/packet"
)

// A complete snapshot repairs lost end reports and repopulates a new Manager.
func (a *App) reportAllSessions() error {
	a.sessionMu.Lock()
	snapshot := packet.SessionSnapshot{ObservedAt: time.Now(), Sessions: make(map[string]int)}
	for id, count := range a.activeSessions {
		if count > 0 {
			snapshot.Sessions[id] = count
		}
	}
	requests := make(map[*http.Request]context.CancelFunc, len(a.sessionCancels))
	for request, cancel := range a.sessionCancels {
		requests[request] = cancel
	}
	a.sessionMu.Unlock()
	for request, cancel := range requests {
		if _, ok := a.getSessionID(request); !ok {
			cancel()
		}
	}
	if a.managerBaseURL == "" {
		return nil
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), a.sessionReportTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.managerBaseURL+"/user/session-snapshot", bytes.NewReader(data))
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
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("session synchronization returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func (a *App) sessionReporter() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		if err := a.reportAllSessions(); err != nil {
			log.Printf("session synchronization: %v", err)
		}
		select {
		case <-a.done:
			return
		case <-ticker.C:
		}
	}
}
