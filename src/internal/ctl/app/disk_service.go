package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/elecbug/linuxus/src/internal/common/config"
	"github.com/elecbug/linuxus/src/internal/common/diskservice"
	"github.com/elecbug/linuxus/src/internal/common/packet"
	"github.com/elecbug/linuxus/src/internal/common/ruleset"
	"github.com/elecbug/linuxus/src/internal/common/user"
)

func (a *App) diskServiceDir() string {
	return filepath.Join(filepath.Dir(a.Config.AuthService.Mounts.HostAuthListPath), ".disk-service")
}
func (a *App) diskSocket() string { return filepath.Join(a.diskServiceDir(), "disk.sock") }

func (a *App) withDiskLock(action func() error) error {
	if err := os.MkdirAll(a.diskServiceDir(), 0700); err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(a.diskServiceDir(), "ensure.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	return action()
}

// Pin the selected configuration for the child, independently of its binary's
// location or any stale LINUXUS_CONFIG inherited from the invoking shell.
func (a *App) diskServiceCommand() *exec.Cmd {
	cmd := exec.Command(a.execPath, "serve-disks")
	cmd.Env = config.WithConfigFile(os.Environ(), a.configFile)
	return cmd
}

// startDiskService uses the same host privileges as the invoking up command.
// It does not grant the Manager any additional Docker or system capabilities.
func (a *App) startDiskService() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("auto-ensure requires root; run linuxusctl up with sudo")
	}
	if err := a.stopDiskService(); err != nil {
		return err
	}
	if err := os.MkdirAll(a.diskServiceDir(), 0700); err != nil {
		return err
	}
	output, err := os.OpenFile(filepath.Join(a.diskServiceDir(), "service.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer output.Close()
	cmd := a.diskServiceCommand()
	cmd.Stdout = output
	cmd.Stderr = output
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start host disk service: %w", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	client := diskservice.NewClient(a.diskSocket(), time.Second)
	defer client.CloseIdleConnections()
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			return fmt.Errorf("disk service exited before startup (%v); see %s", err, output.Name())
		case <-deadline.C:
			_ = cmd.Process.Kill()
			<-done
			return fmt.Errorf("disk service startup timed out; see %s", output.Name())
		case <-ticker.C:
			resp, err := client.Get("http://disk-service/healthz")
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
		}
	}
}

func (a *App) stopDiskService() error {
	info, err := os.Lstat(a.diskSocket())
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("disk service socket path is not a socket")
	}
	client := diskservice.NewClient(a.diskSocket(), 5*time.Second)
	defer client.CloseIdleConnections()
	resp, err := client.Post("http://disk-service/shutdown", "application/json", nil)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if errors.Is(err, syscall.ECONNREFUSED) {
		// A draining service has closed its listener but still owns the lock.
		// Only remove an abandoned socket after the process releases that lock.
		lock, lockErr := os.OpenFile(filepath.Join(a.diskServiceDir(), "service.lock"), os.O_CREATE|os.O_RDWR, 0600)
		if lockErr != nil {
			return lockErr
		}
		defer lock.Close()
		lockErr = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if lockErr == nil {
			defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
			if err := os.Remove(a.diskSocket()); err != nil && !os.IsNotExist(err) {
				return err
			}
			return nil
		}
		if !errors.Is(lockErr, syscall.EWOULDBLOCK) {
			return lockErr
		}
	} else if err != nil {
		return fmt.Errorf("stop disk service: %w", err)
	} else {
		resp.Body.Close()
		if resp.StatusCode != http.StatusAccepted {
			return fmt.Errorf("disk service refused shutdown: %d", resp.StatusCode)
		}
	}
	deadline := time.NewTimer(35 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline.C:
			return fmt.Errorf("disk service shutdown timed out")
		case <-ticker.C:
			if _, err := os.Lstat(a.diskSocket()); os.IsNotExist(err) {
				return nil
			}
		}
	}
}

// ServeDisks is the private host mode launched by up when auto-ensure is true.
// It exposes no TCP listener, command execution, path selection or deletion API.
func (a *App) ServeDisks() error {
	if !a.Config.Volumes.AutoEnsure {
		return fmt.Errorf("VOLUMES_AUTO_ENSURE is disabled")
	}
	if os.Geteuid() != 0 {
		return fmt.Errorf("host disk service requires root for mounting disks")
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return a.serveDisks(ctx)
}

func (a *App) serveDisks(ctx context.Context) error {
	if err := os.MkdirAll(a.diskServiceDir(), 0700); err != nil {
		return err
	}
	if err := os.Chmod(a.diskServiceDir(), 0700); err != nil {
		return err
	}
	lock, err := os.OpenFile(filepath.Join(a.diskServiceDir(), "service.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("disk service is already running: %w", err)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	if info, err := os.Lstat(a.diskSocket()); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return fmt.Errorf("disk service socket path is not a socket")
		}
		if err := os.Remove(a.diskSocket()); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: a.diskSocket(), Net: "unix"})
	if err != nil {
		return err
	}
	listener.SetUnlinkOnClose(false)
	defer listener.Close()
	defer os.Remove(a.diskSocket())
	if err := os.Chmod(a.diskSocket(), 0600); err != nil {
		return err
	}
	stopping := make(chan struct{})
	var once sync.Once
	shutdown := func() { once.Do(func() { close(stopping) }) }
	server := &http.Server{
		Handler:           a.diskServiceHandler(shutdown),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
	}
	finished := make(chan error, 1)
	go func() {
		select {
		case <-ctx.Done():
		case <-stopping:
		}
		// Never exit halfway through formatting or mounting a disk. The caller
		// has its own bounded shutdown wait and can retry if preparation is slow.
		finished <- server.Shutdown(context.Background())
	}()
	err = server.Serve(listener)
	shutdown()
	shutdownErr := <-finished
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return shutdownErr
}

func (a *App) diskServiceHandler(shutdown func()) http.Handler {
	mux := http.NewServeMux()
	var mu sync.Mutex
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("POST /shutdown", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		shutdown()
	})
	mux.HandleFunc("POST /ensure-disk", func(w http.ResponseWriter, r *http.Request) {
		var request packet.UserUpRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 1024)).Decode(&request); err != nil || !ruleset.AllowedUserID(request.UserID) {
			http.Error(w, "invalid user ID", http.StatusBadRequest)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		status := http.StatusInternalServerError
		err := a.withDiskLock(func() error {
			if err := r.Context().Err(); err != nil {
				return err
			}
			// Read registration under the same lock as preparation and cleanup.
			users, err := user.LoadUsers(a.Config.AuthService.Mounts.HostAuthListPath)
			if err != nil {
				return fmt.Errorf("cannot load registered users")
			}
			if !user.ExistsUser(users, request.UserID) {
				status = http.StatusForbidden
				return fmt.Errorf("user is not registered")
			}
			a.UserIDs = users
			return a.ensureDiskUserUnlocked(request.UserID)
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("ensure disk: %v", err), status)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	return mux
}
