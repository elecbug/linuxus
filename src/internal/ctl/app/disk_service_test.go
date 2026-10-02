package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/elecbug/linuxus/src/internal/common/convert"
	"github.com/elecbug/linuxus/src/internal/common/diskservice"
	"github.com/elecbug/linuxus/src/internal/common/packet"
	"github.com/elecbug/linuxus/src/internal/common/system_api"
)

type mountedDiskAPI struct {
	system_api.API
	mu      sync.Mutex
	checked []string
}

func (f *mountedDiskAPI) MkdirAll(string, os.FileMode) error { return nil }
func (f *mountedDiskAPI) IsMountPoint(path string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.checked = append(f.checked, path)
	return true, nil
}

func TestDiskServiceOnlyPreparesRegisteredUsers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AUTH_LIST")
	if err := os.WriteFile(path, []byte("alice:hash\n"), 0600); err != nil {
		t.Fatal(err)
	}
	api := &mountedDiskAPI{}
	a := &App{systemAPI: api}
	a.Config.AuthService.Mounts.HostAuthListPath = path
	a.Config.Volumes.Host.Homes = filepath.Join(dir, "homes")
	a.Config.Volumes.Host.Share = filepath.Join(dir, "share")
	a.Config.Volumes.Host.Readonly = filepath.Join(dir, "readonly")
	a.Config.Volumes.DiskLimit = "8M"
	a.Config.UserService.Limits.User.Disk = "8M"
	socket := filepath.Join(dir, "test.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: a.diskServiceHandler(func() {})}
	defer server.Close()
	go server.Serve(listener)
	client := diskservice.NewClient(socket, time.Second)
	defer client.CloseIdleConnections()
	for _, id := range []string{"../outside", "unknown"} {
		if err := diskservice.Ensure(context.Background(), client, id); err == nil {
			t.Fatalf("accepted %q", id)
		}
	}
	api.mu.Lock()
	count := len(api.checked)
	api.mu.Unlock()
	if count != 0 {
		t.Fatal("invalid users reached disk operations")
	}
	if err := diskservice.Ensure(context.Background(), client, "alice"); err != nil {
		t.Fatal(err)
	}
	api.mu.Lock()
	count = len(api.checked)
	api.mu.Unlock()
	if count != 3 {
		t.Fatalf("expected share, readonly and home checks, got %d", count)
	}
}

type recordingDiskAPI struct {
	system_api.API
	images    map[string]int64
	formatted map[string]int
	mounted   map[string]bool
	failMount string
	detached  int
}

func (f *recordingDiskAPI) MkdirAll(string, os.FileMode) error     { return nil }
func (f *recordingDiskAPI) Chown(string, int, int) error           { return nil }
func (f *recordingDiskAPI) Chmod(string, os.FileMode) error        { return nil }
func (f *recordingDiskAPI) IsMountPoint(path string) (bool, error) { return f.mounted[path], nil }
func (f *recordingDiskAPI) Exists(path string) (bool, error)       { _, ok := f.images[path]; return ok, nil }
func (f *recordingDiskAPI) CreateEmptyFile(path string, size int64) error {
	f.images[path] = size
	return nil
}
func (f *recordingDiskAPI) FormatExt4(path string) error            { f.formatted[path]++; return nil }
func (f *recordingDiskAPI) AttachLoopDevice(string) (string, error) { return "/dev/fake-loop", nil }
func (f *recordingDiskAPI) DetachLoopDevice(string) error           { f.detached++; return nil }
func (f *recordingDiskAPI) Mount(source, target string) error {
	if target == f.failMount {
		return fmt.Errorf("mount failed")
	}
	f.mounted[target] = true
	return nil
}

func TestDiskServiceCreatesReusesAndReloadsUsers(t *testing.T) {
	dir := t.TempDir()
	api := &recordingDiskAPI{images: map[string]int64{}, formatted: map[string]int{}, mounted: map[string]bool{}}
	a := &App{systemAPI: api}
	a.Config.AuthService.Mounts.HostAuthListPath = filepath.Join(dir, "AUTH_LIST")
	a.Config.Volumes.Host.Homes = filepath.Join(dir, "homes")
	a.Config.Volumes.Host.Share = filepath.Join(dir, "share")
	a.Config.Volumes.Host.Readonly = filepath.Join(dir, "readonly")
	a.Config.Volumes.DiskLimit = "8M"
	a.Config.UserService.Limits.User.Disk = "16M"
	a.Config.UserService.Limits.Admin.Disk = "32M"
	a.Config.ManagerService.AdminID = "admin"
	handler := a.diskServiceHandler(func() {})
	writeUsers := func(users string) {
		t.Helper()
		if err := os.WriteFile(a.Config.AuthService.Mounts.HostAuthListPath, []byte(users), 0600); err != nil {
			t.Fatal(err)
		}
	}
	ensure := func(id string, wantStatus int) {
		t.Helper()
		body, _ := json.Marshal(packet.UserUpRequest{UserID: id})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("POST", "/ensure-disk", bytes.NewReader(body)))
		if w.Code != wantStatus {
			t.Fatalf("ensure %s: status=%d body=%s", id, w.Code, w.Body.String())
		}
	}
	writeUsers("alice:hash\n")
	ensure("alice", 200)
	if len(api.images) != 3 || len(api.mounted) != 3 {
		t.Fatalf("missing disks: %+v", api)
	}
	ensure("alice", 200)
	// A host reboot loses mounts, but must never reformat existing disk images.
	api.mounted = map[string]bool{}
	ensure("alice", 200)
	writeUsers("alice:hash\nadmin:hash\n")
	ensure("admin", 200)
	for path, size := range map[string]string{
		a.Config.Volumes.Host.Share + ".img":                    "8M",
		a.Config.Volumes.Host.Readonly + ".img":                 "8M",
		filepath.Join(a.Config.Volumes.Host.Homes, "alice.img"): "16M",
		filepath.Join(a.Config.Volumes.Host.Homes, "admin.img"): "32M",
	} {
		want, err := convert.BytesFromString(size)
		if err != nil {
			t.Fatal(err)
		}
		if api.images[path] != want || api.formatted[path] != 1 {
			t.Fatalf("wrong size or reformatted %s: %+v", path, api)
		}
	}
	writeUsers("admin:hash\n")
	ensure("alice", 403)
	writeUsers("admin:hash\nbob:hash\n")
	api.failMount = filepath.Join(a.Config.Volumes.Host.Homes, "bob")
	ensure("bob", 500)
	if api.detached != 1 {
		t.Fatal("failed mount leaked a loop device")
	}
	api.failMount = ""
	ensure("bob", 200)
	if api.formatted[filepath.Join(a.Config.Volumes.Host.Homes, "bob.img")] != 1 {
		t.Fatal("retry reformatted disk")
	}
}

func TestDiskServiceLifecycle(t *testing.T) {
	// Keep the Unix socket path below the operating system's length limit.
	dir, err := os.MkdirTemp("", "linuxus-service-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	a := &App{}
	a.Config.AuthService.Mounts.HostAuthListPath = filepath.Join(dir, "AUTH_LIST")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- a.serveDisks(ctx) }()
	client := diskservice.NewClient(a.diskSocket(), time.Second)
	defer client.CloseIdleConnections()
	deadline := time.Now().Add(3 * time.Second)
	for {
		resp, err := client.Get("http://disk-service/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("disk service did not become healthy")
		}
		time.Sleep(10 * time.Millisecond)
	}
	for path, mode := range map[string]os.FileMode{a.diskServiceDir(): 0700, a.diskSocket(): 0600} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != mode {
			t.Fatalf("%s mode=%v", path, info.Mode())
		}
	}
	// A duplicate process must not steal or unlink the live socket.
	if err := a.serveDisks(ctx); err == nil {
		t.Fatal("duplicate service accepted")
	}
	if err := a.stopDiskService(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("disk service did not stop")
	}
	if _, err := os.Lstat(a.diskSocket()); !os.IsNotExist(err) {
		t.Fatalf("socket remains: %v", err)
	}
	if err := a.stopDiskService(); err != nil {
		t.Fatalf("repeated shutdown: %v", err)
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: a.diskSocket(), Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	listener.SetUnlinkOnClose(false)
	listener.Close()
	if err := a.stopDiskService(); err != nil {
		t.Fatalf("stale socket: %v", err)
	}
	if err := os.WriteFile(a.diskSocket(), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := a.stopDiskService(); err == nil {
		t.Fatal("accepted regular file at socket path")
	}
	if data, err := os.ReadFile(a.diskSocket()); err != nil || string(data) != "keep" {
		t.Fatal("removed non-socket file")
	}
}

type blockingDiskAPI struct {
	mountedDiskAPI
	once             sync.Once
	entered, release chan struct{}
}

func (f *blockingDiskAPI) IsMountPoint(path string) (bool, error) {
	f.once.Do(func() { close(f.entered); <-f.release })
	return f.mountedDiskAPI.IsMountPoint(path)
}

func TestDiskServiceShutdownWaitsForActivePreparation(t *testing.T) {
	dir, err := os.MkdirTemp("", "linuxus-drain-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	api := &blockingDiskAPI{entered: make(chan struct{}), release: make(chan struct{})}
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(api.release) }) }
	defer release()
	a := &App{systemAPI: api}
	a.Config.AuthService.Mounts.HostAuthListPath = filepath.Join(dir, "AUTH_LIST")
	a.Config.Volumes.Host.Homes = filepath.Join(dir, "homes")
	a.Config.Volumes.Host.Share = filepath.Join(dir, "share")
	a.Config.Volumes.Host.Readonly = filepath.Join(dir, "readonly")
	a.Config.Volumes.DiskLimit = "8M"
	a.Config.UserService.Limits.User.Disk = "8M"
	if err := os.WriteFile(a.Config.AuthService.Mounts.HostAuthListPath, []byte("alice:hash\n"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- a.serveDisks(ctx) }()
	client := diskservice.NewClient(a.diskSocket(), 5*time.Second)
	defer client.CloseIdleConnections()
	deadline := time.Now().Add(3 * time.Second)
	for {
		resp, err := client.Get("http://disk-service/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("service startup timed out")
		}
		time.Sleep(10 * time.Millisecond)
	}
	prepared := make(chan error, 1)
	go func() { prepared <- diskservice.Ensure(context.Background(), client, "alice") }()
	select {
	case <-api.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("preparation did not start")
	}
	cancel()
	// Wait until the listener closes while the preparation request is draining.
	deadline = time.Now().Add(3 * time.Second)
	for {
		resp, err := client.Get("http://disk-service/healthz")
		if err != nil {
			break
		}
		resp.Body.Close()
		if time.Now().After(deadline) {
			t.Fatal("listener did not close")
		}
		time.Sleep(10 * time.Millisecond)
	}
	stopped := make(chan error, 1)
	go func() { stopped <- a.stopDiskService() }()
	select {
	case err := <-stopped:
		t.Fatalf("stop returned before preparation finished: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if _, err := os.Stat(a.diskSocket()); err != nil {
		t.Fatalf("draining socket removed: %v", err)
	}
	release()
	for name, result := range map[string]<-chan error{"prepare": prepared, "serve": done, "stop": stopped} {
		select {
		case err := <-result:
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("%s timed out", name)
		}
	}
}
