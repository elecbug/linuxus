package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserUpUsesExactAddressForCaseDistinctAccounts(t *testing.T) {
	addresses := map[string]string{"alice": "10.10.0.2", "Alice": "10.10.0.18"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			UserID string `json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "user_id": request.UserID,
			"container_name": "user_" + request.UserID, "container_ip": addresses[request.UserID]})
	}))
	defer server.Close()
	a := &App{managerBaseURL: server.URL, managerClient: server.Client()}
	for id, want := range addresses {
		host, err := a.ensureUserContainerReady(context.Background(), id)
		if err != nil || host != want {
			t.Fatalf("user=%s backend=%s want=%s error=%v", id, host, want, err)
		}
	}
}

func TestUserUpRejectsMissingOrInvalidTargetAddresses(t *testing.T) {
	for _, address := range []string{"", "user_alice", "10.10.0.2:7681", "10.10.0.2/path", "::1"} {
		t.Run(address, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(map[string]any{"ok": true, "user_id": "alice", "container_name": "user_alice", "container_ip": address})
			}))
			defer server.Close()
			a := &App{managerBaseURL: server.URL, managerClient: server.Client()}
			if _, err := a.ensureUserContainerReady(context.Background(), "alice"); err == nil {
				t.Fatal("accepted an ambiguous or invalid runtime address")
			}
		})
	}
}

func TestUserUpRejectsMismatchedUserIdentity(t *testing.T) {
	for _, id := range []string{"", "Alice", "bob"} {
		t.Run(id, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(map[string]any{"ok": true, "user_id": id, "container_ip": "10.10.0.2"})
			}))
			defer server.Close()
			a := &App{managerBaseURL: server.URL, managerClient: server.Client()}
			if _, err := a.ensureUserContainerReady(context.Background(), "alice"); err == nil {
				t.Fatal("accepted another account's runtime address")
			}
		})
	}
}

func TestUserUpSendsManagerSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/user/up" || r.Header.Get("X-Manager-Session-Secret") != "test-secret" {
			t.Error("missing manager authentication")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"user_id":"alice","container_name":"user_alice","container_ip":"10.10.0.2"}`))
	}))
	defer server.Close()
	a := &App{managerBaseURL: server.URL, managerClient: server.Client(), managerSessionSecret: "test-secret"}
	host, err := a.ensureUserContainerReady(context.Background(), "alice")
	if err != nil || host != "10.10.0.2" {
		t.Fatalf("backend=%s error=%v", host, err)
	}
}
