package handler

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/elecbug/linuxus/src/internal/common/diskservice"
	"github.com/elecbug/linuxus/src/internal/common/packet"
	"github.com/elecbug/linuxus/src/internal/manager/config"
)

type runtimeDocker struct {
	mu              sync.Mutex
	exists, running bool
	mutations       []string
	hostConfig      container.HostConfig
	beforeMutation  func()
}

func (d *runtimeDocker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.mu.Lock()
	defer d.mu.Unlock()
	path := strings.TrimPrefix(r.URL.Path, "/v1.47")
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodPost {
		if d.beforeMutation != nil {
			d.beforeMutation()
		}
		d.mutations = append(d.mutations, path)
	}
	switch {
	case path == "/images/test-user/json":
		json.NewEncoder(w).Encode(map[string]any{"Id": "image-id"})
	case path == "/containers/user_alice/json":
		if !d.exists {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"message": "No such container"})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"Id": "user-id", "State": map[string]bool{"Running": d.running},
			"NetworkSettings": map[string]any{"Networks": map[string]any{"net_alice": map[string]string{"IPAddress": "10.10.0.2"}}},
		})
	case path == "/networks":
		w.Write([]byte("[]"))
	case path == "/networks/create":
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"Id":"network-id"}`))
	case path == "/networks/net_alice":
		w.Write([]byte(`{"IPAM":{"Config":[{"Subnet":"10.10.0.0/24"}]},"Containers":{"auth-id":{"Name":"auth"}}}`))
	case path == "/containers/create":
		var body struct{ HostConfig container.HostConfig }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		d.hostConfig = body.HostConfig
		d.exists = true
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"Id":"user-id"}`))
	case path == "/containers/user-id/start" || path == "/containers/user_alice/start":
		d.running = true
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "unexpected Docker API: "+r.Method+" "+path, http.StatusInternalServerError)
	}
}

func runtimeTestServer(t *testing.T, d *runtimeDocker, diskHandler http.HandlerFunc) *Server {
	t.Helper()
	dockerHTTP := httptest.NewServer(d)
	t.Cleanup(dockerHTTP.Close)
	cli, err := client.NewClientWithOpts(client.WithHost("tcp://"+strings.TrimPrefix(dockerHTTP.URL, "http://")), client.WithVersion("1.47"), client.WithHTTPClient(dockerHTTP.Client()))
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp("", "linuxus-manager-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket := filepath.Join(dir, "disk.sock")
	if diskHandler != nil {
		listener, err := net.Listen("unix", socket)
		if err != nil {
			t.Fatal(err)
		}
		httpServer := &http.Server{Handler: diskHandler}
		t.Cleanup(func() { httpServer.Close() })
		go httpServer.Serve(listener)
	}
	s := &Server{docker: cli, diskClient: diskservice.NewClient(socket, time.Second), cfg: &config.Config{
		AutoEnsure: true, UserImage: "test-user", UserContainerNamePrefix: "user_", NetworkPrefix: "net_",
		BaseIP: "10.10.0.0", AuthContainerName: "auth", AdminUserID: "admin", ManagerWaitTime: time.Second,
		HostHomesDir: "/volumes/homes", HostShareDir: "/volumes/share", HostReadonlyDir: "/volumes/readonly",
		ContainerRuntimeUser: "user", ContainerShareDir: "/home/share", ContainerReadonlyDir: "/home/readonly",
	}}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestAutoEnsureRuntime(t *testing.T) {
	for _, tc := range []struct {
		name                                 string
		exists, running, enabled             bool
		diskStatus, wantCalls, wantMutations int
		wantError                            bool
	}{
		{"new", false, false, true, 200, 1, 3, false},
		{"stopped", true, false, true, 200, 1, 1, false},
		{"running", true, true, true, 500, 0, 0, false},
		{"disabled", false, false, false, 500, 0, 3, false},
		{"disk_failure", false, false, true, 500, 1, 0, true},
		{"stopped_disk_failure", true, false, true, 500, 1, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			d := &runtimeDocker{exists: tc.exists, running: tc.running, beforeMutation: func() {
				if tc.enabled && calls.Load() == 0 {
					t.Error("Docker mutated before disk preparation")
				}
			}}
			s := runtimeTestServer(t, d, func(w http.ResponseWriter, r *http.Request) {
				var request packet.UserUpRequest
				if r.Method != "POST" || r.URL.Path != "/ensure-disk" {
					t.Error("invalid disk request")
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.UserID != "alice" {
					t.Error("invalid user")
				}
				calls.Add(1)
				w.WriteHeader(tc.diskStatus)
			})
			s.cfg.AutoEnsure = tc.enabled
			resp, err := s.ensureUserRuntimeReady(context.Background(), "alice")
			if (err != nil) != tc.wantError {
				t.Fatalf("response=%v error=%v", resp, err)
			}
			if !tc.wantError && (resp == nil || !resp.OK) {
				t.Fatalf("not ready: %v", resp)
			}
			if got := int(calls.Load()); got != tc.wantCalls {
				t.Fatalf("disk calls=%d, want %d", got, tc.wantCalls)
			}
			d.mu.Lock()
			defer d.mu.Unlock()
			if len(d.mutations) != tc.wantMutations {
				t.Fatalf("Docker mutations: %v", d.mutations)
			}
			if !tc.exists && !tc.wantError {
				if len(d.hostConfig.Binds) != 0 || len(d.hostConfig.Mounts) != 3 {
					t.Fatalf("unsafe binds: %+v", d.hostConfig)
				}
				for _, m := range d.hostConfig.Mounts {
					if m.Type != mount.TypeBind || (m.BindOptions != nil && m.BindOptions.CreateMountpoint) {
						t.Fatalf("mount creates missing source: %+v", m)
					}
				}
				if !d.hostConfig.Mounts[2].ReadOnly {
					t.Fatal("regular user can write readonly share")
				}
			}
		})
	}
}

func TestAutoEnsureUnavailableAndTimeoutPreventStart(t *testing.T) {
	for _, timeout := range []bool{false, true} {
		t.Run(map[bool]string{false: "unavailable", true: "timeout"}[timeout], func(t *testing.T) {
			d := &runtimeDocker{}
			var handler http.HandlerFunc
			if timeout {
				handler = func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }
			}
			s := runtimeTestServer(t, d, handler)
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			if _, err := s.ensureUserRuntimeReady(ctx, "alice"); err == nil {
				t.Fatal("expected disk service error")
			}
			d.mu.Lock()
			defer d.mu.Unlock()
			if len(d.mutations) != 0 {
				t.Fatalf("Docker changed after disk failure: %v", d.mutations)
			}
		})
	}
}

func TestConcurrentUserUpPreparesOnce(t *testing.T) {
	var calls atomic.Int32
	d := &runtimeDocker{}
	s := runtimeTestServer(t, d, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(200) })
	var wg sync.WaitGroup
	for range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if _, err := s.ensureUserRuntimeReady(ctx, "alice"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("prepared disk %d times", calls.Load())
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.mutations) != 3 {
		t.Fatalf("created duplicate runtime: %v", d.mutations)
	}
}

func TestUserUpRequiresSharedSecret(t *testing.T) {
	s := &Server{cfg: &config.Config{ManagerSessionSecret: "expected"}}
	for _, secret := range []string{"", "incorrect"} {
		r := httptest.NewRequest("POST", "/user/up", strings.NewReader(`{"user_id":"alice"}`))
		r.Header.Set("X-Manager-Session-Secret", secret)
		w := httptest.NewRecorder()
		s.HandleUserUp(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d", w.Code)
		}
	}
}
