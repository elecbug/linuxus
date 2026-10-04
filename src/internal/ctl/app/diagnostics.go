package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/docker/docker/client"
	"github.com/elecbug/linuxus/src/internal/common/config"
	"github.com/elecbug/linuxus/src/internal/common/diskservice"
)

func checkedSettings(configFile string) (*App, error) {
	a := &App{configFile: configFile}
	if err := a.loadSettings(); err != nil {
		return nil, err
	}
	if err := config.ValidateConfig(&a.Config); err != nil {
		return nil, err
	}
	return a, nil
}

// CheckConfig validates settings without a Docker client or credential reads.
// Only paths are printed; session secrets and account hashes stay private.
func CheckConfig(configFile string, output io.Writer) error {
	a, err := checkedSettings(configFile)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "Configuration valid: %s\nAuth list: %s\nVolumes: %s\nHomes: %s\nShare: %s\nReadonly: %s\nAuto-ensure: %t\nAdmin URL: /%s\n",
		a.configFile, a.Config.AuthService.Mounts.HostAuthListPath,
		a.Config.Volumes.Host.Volumes, a.Config.Volumes.Host.Homes,
		a.Config.Volumes.Host.Share, a.Config.Volumes.Host.Readonly, a.Config.Volumes.AutoEnsure, a.Config.AdminRoute())
	return err
}

type doctorChecks struct {
	goos     string
	euid     int
	lookPath func(string) (string, error)
	docker   func(context.Context) error
	disk     func(context.Context, string) error
	runtime  func(*App, func(string, string, string))
}

// Doctor reports independent prerequisite failures without changing host state.
func Doctor(configFile string, output io.Writer) error {
	return runDoctor(configFile, output, doctorChecks{
		goos: runtime.GOOS, euid: os.Geteuid(), lookPath: exec.LookPath,
		docker: probeDocker, disk: probeDiskService,
		runtime: func(a *App, report func(string, string, string)) { a.diagnoseRuntime(report) },
	})
}

func runDoctor(configFile string, output io.Writer, checks doctorChecks) error {
	failures, warnings := 0, 0
	var writeErr error
	report := func(status, name, detail string) {
		switch status {
		case "FAIL":
			failures++
		case "WARN":
			warnings++
		}
		if writeErr == nil {
			_, writeErr = fmt.Fprintf(output, "[%s] %s: %s\n", status, name, detail)
		}
	}
	a, err := checkedSettings(configFile)
	if err != nil {
		report("FAIL", "configuration", err.Error())
	} else {
		report("OK", "configuration", a.configFile)
		info, err := os.Stat(a.configFile)
		if err != nil {
			report("FAIL", "configuration permissions", err.Error())
		} else if info.Mode().Perm()&0077 != 0 {
			report("WARN", "configuration permissions", "other users can access settings; use chmod 600 on the configuration file")
		}
		paths := []struct {
			name, path string
		}{
			{"auth list", a.Config.AuthService.Mounts.HostAuthListPath},
			{"volumes", a.Config.Volumes.Host.Volumes},
			{"homes", a.Config.Volumes.Host.Homes},
			{"share", a.Config.Volumes.Host.Share},
			{"readonly", a.Config.Volumes.Host.Readonly},
		}
		for _, entry := range paths {
			info, err := os.Stat(entry.path)
			switch {
			case os.IsNotExist(err):
				report("WARN", entry.name, entry.path+" does not exist yet; up prepares missing storage")
			case err != nil:
				report("FAIL", entry.name, err.Error())
			case entry.name == "auth list" && info.Mode().Perm()&0077 != 0:
				report("WARN", entry.name, entry.path+" is accessible to other users; use chmod 600")
			default:
				report("OK", entry.name, entry.path)
			}
		}
	}
	if checks.goos != "linux" {
		report("FAIL", "host OS", "disk operations require Linux")
	} else {
		report("OK", "host OS", "Linux")
	}
	if checks.euid != 0 {
		report("WARN", "privileges", "disk operations require root; run up and ensure-disk with sudo")
	} else {
		report("OK", "privileges", "root (mount capability is required separately in restricted environments)")
	}
	for _, name := range []string{"mkfs.ext4", "losetup"} {
		path, err := checks.lookPath(name)
		if err != nil {
			report("FAIL", name, "executable not found in PATH")
		} else {
			report("OK", name, path)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = checks.docker(ctx)
	cancel()
	if err != nil {
		report("FAIL", "Docker", err.Error())
	} else {
		report("OK", "Docker", "daemon reachable")
	}
	if a != nil {
		if a.Config.Volumes.AutoEnsure {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			err := checks.disk(ctx, a.diskSocket())
			cancel()
			if err != nil {
				report("FAIL", "auto-ensure", fmt.Sprintf("%v; run sudo linuxusctl up and inspect %s/service.log", err, a.diskServiceDir()))
			} else {
				report("OK", "auto-ensure", "host disk service reachable")
			}
		} else {
			report("OK", "auto-ensure", "disabled; new users require ensure-disk")
		}
	}
	if a != nil && checks.runtime != nil {
		checks.runtime(a, report)
	}
	if writeErr != nil {
		return writeErr
	}
	if _, err := fmt.Fprintf(output, "Checks complete: %d failure(s), %d warning(s).\n", failures, warnings); err != nil {
		return err
	}
	if failures > 0 {
		return fmt.Errorf("doctor found %d failed check(s)", failures)
	}
	return nil
}

func probeDocker(ctx context.Context) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()
	_, err = cli.Ping(ctx)
	return err
}

func probeDiskService(ctx context.Context, socket string) error {
	cli := diskservice.NewClient(socket, 2*time.Second)
	defer cli.CloseIdleConnections()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://disk-service/healthz", nil)
	if err != nil {
		return err
	}
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned HTTP %d", resp.StatusCode)
	}
	return nil
}
