package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/client"
	"github.com/elecbug/linuxus/src/internal/ctl/log"
)

func systemdQuote(value string) (string, error) {
	if strings.ContainsAny(value, "\x00\r\n") {
		return "", fmt.Errorf("systemd paths must not contain line breaks")
	}
	value = strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "%", "%%", "$", "$$").Replace(value)
	return "\"" + value + "\"", nil
}

// WriteSystemdUnit generates a reviewable unit; it does not install or enable it.
func WriteSystemdUnit(executable, configFile string, output io.Writer) error {
	if !filepath.IsAbs(executable) || !filepath.IsAbs(configFile) {
		return fmt.Errorf("systemd paths must be absolute")
	}
	binary, err := systemdQuote(executable)
	if err != nil {
		return err
	}
	// Environment= does not expand dollars, unlike ExecStart=.
	setting, err := systemdQuote("LINUXUS_CONFIG=" + configFile)
	if err != nil {
		return err
	}
	setting = strings.ReplaceAll(setting, "$$", "$")
	_, err = fmt.Fprintf(output, `[Unit]
Description=Linuxus runtime supervisor
Requires=docker.service
After=docker.service local-fs.target network-online.target
Wants=network-online.target
StartLimitIntervalSec=0

[Service]
Type=simple
User=root
UMask=0077
Environment=%s
ExecStart=%s supervise
Restart=on-failure
RestartSec=5s
TimeoutStopSec=120s
KillMode=control-group

[Install]
WantedBy=multi-user.target
`, setting, binary)
	return err
}

func (a *App) supervisorLock() string { return filepath.Join(a.diskServiceDir(), "supervisor.lock") }

func (a *App) checkSupervisor() error {
	if a.supervised {
		return nil
	}
	file, err := os.OpenFile(a.supervisorLock(), os.O_RDWR, 0600)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("a supervisor is active; use systemctl stop/restart linuxus instead")
	}
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}

func (a *App) ensureCachedImages() error {
	for _, name := range []string{a.authImageName(), a.managerImageName(), a.userImageName()} {
		if _, err := a.dockerClient.ImageInspect(a.context, name, client.ImageInspectWithRawResponse(nil)); err != nil {
			if !errdefs.IsNotFound(err) {
				return err
			}
			return a.buildRuntimeImages()
		}
	}
	return nil
}

// Supervise mounts disks before starting services, then recovers failed host
// and container processes. systemd owns this process and all of its children.
func (a *App) Supervise() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("supervise requires root")
	}
	if err := os.MkdirAll(a.diskServiceDir(), 0700); err != nil {
		return err
	}
	file, err := os.OpenFile(a.supervisorLock(), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("another supervisor is active: %w", err)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	a.supervised = true
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	a.context = ctx
	defer func() {
		shutdown, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		a.context = shutdown
		if err := a.ServiceDown(nil); err != nil {
			log.Log(log.ERROR_PREFIX, "supervisor shutdown: %v", err)
		}
	}()
	if err := a.ServiceUp(nil); err != nil {
		return err
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			probe, cancel := context.WithTimeout(ctx, 2*time.Second)
			err := probeDiskService(probe, a.diskSocket())
			cancel()
			if err != nil {
				if err := a.startDiskService(); err != nil {
					return fmt.Errorf("recover disk service: %w", err)
				}
			}
			for _, name := range []string{a.Config.ManagerService.Container.Name, a.Config.AuthService.Container.Name} {
				probe, cancel := context.WithTimeout(ctx, 5*time.Second)
				info, err := a.dockerClient.ContainerInspect(probe, name)
				cancel()
				if err != nil && !errdefs.IsNotFound(err) {
					return err
				}
				if err == nil && info.State != nil && info.State.Running {
					continue
				}
				if name == a.Config.ManagerService.Container.Name {
					err = a.ensureManagerContainer()
				} else {
					err = a.ensureAuthContainer()
				}
				if err != nil {
					return fmt.Errorf("recover %s: %w", name, err)
				}
			}
		}
	}
}

func (a *App) serviceRestartPolicy() string {
	if a.supervised {
		return "no"
	}
	return "unless-stopped"
}
