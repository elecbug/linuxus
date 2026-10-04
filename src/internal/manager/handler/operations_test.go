package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/client"
	commonconfig "github.com/elecbug/linuxus/src/internal/common/config"
	"github.com/elecbug/linuxus/src/internal/common/user"
	"github.com/elecbug/linuxus/src/internal/manager/config"
)

func TestLockedAccountCannotResumeExistingRuntime(t *testing.T) {
	d := &runtimeDocker{exists: true, running: true}
	s := runtimeTestServer(t, d, nil)
	path := filepath.Join(t.TempDir(), "AUTH_LIST")
	os.WriteFile(path, []byte("alice:hash\n"), 0600)
	user.UpdateAccount(path, "alice", "lock", "")
	s.cfg.AuthListFile = path
	if _, err := s.ensureUserRuntimeReady(context.Background(), "alice"); err == nil {
		t.Fatal("locked user resumed a running container")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.mutations) != 0 {
		t.Fatal("rejected user changed Docker")
	}
}

func TestPreparationQueueLimitAndCancellation(t *testing.T) {
	s := &Server{cfg: &config.Config{MaxPending: 1}}
	s.prepareMu.Lock()
	defer s.prepareMu.Unlock()
	s.pending.Store(1)
	if _, err := s.ensureUserRuntimeReady(context.Background(), "alice"); err == nil || !strings.Contains(err.Error(), "queue") {
		t.Fatal(err)
	}
	if s.pending.Load() != 1 {
		t.Fatal("queue counter leaked")
	}
	s.pending.Store(0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.ensureUserRuntimeReady(ctx, "alice"); err == nil {
		t.Fatal("ignored cancellation")
	}
	if s.pending.Load() != 0 {
		t.Fatal("cancelled request occupied queue")
	}
}

func TestSnapshotRepairsLostEndReport(t *testing.T) {
	s := &Server{cfg: &config.Config{ManagerSessionSecret: "secret"}, runtimes: map[string]*RuntimeState{"alice": {UserID: "alice", ActiveSessions: 1, LastObservedAt: time.Now().Add(-time.Minute)}}}
	r := httptest.NewRequest("POST", "/user/session-snapshot", strings.NewReader(`{"observed_at":"`+time.Now().UTC().Format(time.RFC3339Nano)+`","sessions":{}}`))
	r.Header.Set("X-Manager-Session-Secret", "secret")
	w := httptest.NewRecorder()
	s.HandleSessionSnapshot(w, r)
	if w.Code != http.StatusOK || s.runtimes["alice"].ActiveSessions != 0 || s.runtimes["alice"].IdleSince.IsZero() {
		t.Fatal("lost end report not repaired")
	}
}

func TestTemplateSelectionAndClassOverride(t *testing.T) {
	d := &runtimeDocker{}
	s := runtimeTestServer(t, d, nil)
	path := filepath.Join(t.TempDir(), "AUTH_LIST")
	os.WriteFile(path, []byte("alice:hash\n"), 0600)
	s.cfg.AuthListFile = path
	s.cfg.Templates = map[string]commonconfig.Template{"python": {Image: "classroom/python:v1", Seed: "/opt/python"}}
	s.cfg.Classes = map[string]string{"class-a": "python"}
	user.UpdateAccount(path, "alice", "class", "class-a")
	name, profile, err := s.userTemplate("alice")
	if err != nil || name != "python" || profile.Image != "classroom/python:v1" {
		t.Fatal(name, profile, err)
	}
	user.UpdateAccount(path, "alice", "template", "default")
	name, profile, err = s.userTemplate("alice")
	if err != nil || name != "default" || profile.Image != "test-user" {
		t.Fatal("direct default template ignored")
	}
}

func TestRunningLimitAndReconciliationUseDockerState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"Id":"alice","Names":["/user_alice"],"State":"running"},{"Id":"other","Names":["/unrelated"],"State":"running"}]`))
	}))
	defer server.Close()
	cli, err := client.NewClientWithOpts(client.WithHost("tcp://"+strings.TrimPrefix(server.URL, "http://")), client.WithVersion("1.47"), client.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()
	s := &Server{docker: cli, cfg: &config.Config{UserContainerNamePrefix: "user_", MaxRunning: 1}}
	if err := s.checkAdmission(context.Background()); err == nil {
		t.Fatal("running limit ignored")
	}
	s.cfg.MaxRunning = 2
	if err := s.checkAdmission(context.Background()); err != nil {
		t.Fatal("unrelated container counted", err)
	}
	if err := s.reconcileRuntimes(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(s.runtimes) != 1 || time.Until(s.runtimes["alice"].IdleSince) < 80*time.Second {
		t.Fatal("restart grace missing")
	}
	s.updateSessionState("alice", 1, time.Now())
	s.reconcileRuntimes(context.Background())
	if s.runtimes["alice"].ActiveSessions != 1 {
		t.Fatal("discovery overwrote live session state")
	}
}

type statusTransport func(*http.Request) (*http.Response, error)

func (f statusTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAdminStatusReportsDiskServiceFailures(t *testing.T) {
	for _, mode := range []string{"unavailable", "http-error", "bad-json", "null", "success"} {
		t.Run(mode, func(t *testing.T) {
			docker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`[{"Id":"alice","Names":["/user_alice"],"State":"running"}]`))
			}))
			defer docker.Close()
			cli, err := client.NewClientWithOpts(client.WithHost("tcp://"+strings.TrimPrefix(docker.URL, "http://")), client.WithVersion("1.47"), client.WithHTTPClient(docker.Client()))
			if err != nil {
				t.Fatal(err)
			}
			defer cli.Close()
			s := &Server{docker: cli, cfg: &config.Config{UserContainerNamePrefix: "user_", ManagerSessionSecret: "secret"}}
			if mode != "unavailable" {
				s.diskClient = &http.Client{Transport: statusTransport(func(r *http.Request) (*http.Response, error) {
					status, body := 200, `{"alice":{"mounted":true,"used_bytes":123}}`
					switch mode {
					case "http-error":
						status, body = 503, "busy"
					case "bad-json":
						body = "broken JSON"
					case "null":
						body = "null"
					}
					return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
				})}
			}
			r := httptest.NewRequest(http.MethodGet, "/admin/status", nil)
			r.Header.Set("X-Manager-Session-Secret", "secret")
			w := httptest.NewRecorder()
			s.HandleAdminStatus(w, r)
			var rows map[string]map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil || w.Code != 200 {
				t.Fatalf("status response failed: %d %s %v", w.Code, w.Body, err)
			}
			if rows["alice"]["state"] != "running" {
				t.Fatal("disk failure discarded container status")
			}
			warned := w.Header().Get("X-Linuxus-Disk-Error") != ""
			if warned != (mode != "success") || (rows["alice"]["disk_error"] != nil) != warned {
				t.Fatalf("incorrect disk failure indication: %s %+v", w.Header(), rows)
			}
		})
	}
}
