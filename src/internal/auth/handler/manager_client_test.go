package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserUpSendsManagerSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/user/up" || r.Header.Get("X-Manager-Session-Secret") != "test-secret" {
			t.Error("missing manager authentication")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"container_name":"user_alice"}`))
	}))
	defer server.Close()
	a := &App{managerBaseURL: server.URL, managerClient: server.Client(), managerSessionSecret: "test-secret"}
	name, err := a.ensureUserContainerReady(context.Background(), "alice")
	if err != nil || name != "user_alice" {
		t.Fatalf("container=%s error=%v", name, err)
	}
}
