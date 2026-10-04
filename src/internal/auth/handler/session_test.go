package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/elecbug/linuxus/src/internal/common/packet"
)

func TestLogoutPreservesOtherOpenTerminals(t *testing.T) {
	a := &App{activeSessions: map[string]int{"alice": 2}, sessionKey: []byte("test-secret"), loginPath: "login"}
	cookie := httptest.NewRecorder()
	a.setSessionCookie(cookie, httptest.NewRequest("POST", "/login", nil), "alice", "test-hash")
	r := httptest.NewRequest("GET", "/logout", nil)
	r.AddCookie(cookie.Result().Cookies()[0])
	w := httptest.NewRecorder()
	a.handleLogout(w, r)
	if a.activeSessions["alice"] != 2 {
		t.Fatal("logout erased sessions that are still connected")
	}
	if w.Code != http.StatusSeeOther || len(w.Result().Cookies()) != 1 || w.Result().Cookies()[0].MaxAge >= 0 {
		t.Fatal("logout did not clear the browser cookie")
	}
}

func TestSessionSnapshotsKeepEventOrder(t *testing.T) {
	reports := make(chan packet.SessionStateReport, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var report packet.SessionStateReport
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			t.Error(err)
			return
		}
		reports <- report
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	a := &App{activeSessions: map[string]int{}, managerBaseURL: server.URL, managerClient: server.Client(), sessionReportTimeout: time.Second}
	a.markSessionStart("alice")
	if a.activeSessions["alice"] != 1 {
		t.Fatal("start count was deferred")
	}
	endBoundary := time.Now()
	a.markSessionEnd("alice")
	if a.activeSessions["alice"] != 0 {
		t.Fatal("end count was deferred")
	}
	var start, end packet.SessionStateReport
	for range 2 {
		select {
		case report := <-reports:
			if report.ActiveSessions == 1 {
				start = report
			} else {
				end = report
			}
		case <-time.After(2 * time.Second):
			t.Fatal("missing session report")
		}
	}
	if start.ObservedAt.IsZero() || end.ObservedAt.IsZero() || start.ObservedAt.After(endBoundary) || end.ObservedAt.Before(endBoundary) {
		t.Fatalf("reports lost observation time: start=%v boundary=%v end=%v", start, endBoundary, end)
	}
}
