package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/elecbug/linuxus/src/internal/common/config"
)

func diagnosticConfig(t *testing.T, autoEnsure bool) string {
	t.Helper()
	root := t.TempDir()
	text := strings.ReplaceAll(config.DefaultEnv, "/var/lib/linuxus/", "./state/")
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "VOLUMES_AUTO_ENSURE=") {
			lines[i] = "VOLUMES_AUTO_ENSURE=false"
			if autoEnsure {
				lines[i] = "VOLUMES_AUTO_ENSURE=true"
			}
		}
	}
	path := filepath.Join(root, ".env")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestConfigCheckIgnoresDockerAndAccountContents(t *testing.T) {
	path := diagnosticConfig(t, false)
	auth := filepath.Join(filepath.Dir(path), "state", "data", "AUTH_LIST")
	if err := os.MkdirAll(filepath.Dir(auth), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(auth, []byte("invalid private account contents"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOCKER_HOST", "invalid://no-docker")
	t.Chdir(t.TempDir())
	var output bytes.Buffer
	if err := CheckConfig(path, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), auth) || strings.Contains(output.String(), "private account") {
		t.Fatal("configuration check did not print resolved paths safely")
	}
	a, err := checkedSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{a.Config.AuthService.Security.SessionSecret, a.Config.ManagerService.Security.SessionSecret} {
		if strings.Contains(output.String(), secret) {
			t.Fatal("configuration check printed a secret")
		}
	}
	if _, err := os.Stat(a.Config.Volumes.Host.Volumes); !os.IsNotExist(err) {
		t.Fatal("configuration check created storage")
	}
}

func TestDoctorAggregatesFailuresAndDoesNotCreateStorage(t *testing.T) {
	path := diagnosticConfig(t, true)
	var output bytes.Buffer
	var probes []string
	err := runDoctor(path, &output, doctorChecks{
		goos: "linux", euid: 1000,
		lookPath: func(name string) (string, error) {
			probes = append(probes, name)
			return "", errors.New("missing")
		},
		docker: func(ctx context.Context) error {
			probes = append(probes, "Docker")
			deadline, ok := ctx.Deadline()
			if !ok || time.Until(deadline) > 5*time.Second {
				t.Fatal("Docker probe has no bounded deadline")
			}
			return errors.New("daemon unavailable")
		},
		disk: func(ctx context.Context, socket string) error {
			probes = append(probes, "disk")
			if !strings.HasPrefix(socket, filepath.Dir(path)) {
				t.Fatal("disk service probe ignored selected configuration")
			}
			deadline, ok := ctx.Deadline()
			if !ok || time.Until(deadline) > 2*time.Second {
				t.Fatal("disk probe has no bounded deadline")
			}
			return errors.New("not started")
		},
	})
	if err == nil || !strings.Contains(err.Error(), "4 failed") || len(probes) != 4 {
		t.Fatalf("not all checks ran: %v, %v", probes, err)
	}
	if !strings.Contains(output.String(), "[WARN] privileges") || !strings.Contains(output.String(), "service.log") {
		t.Fatalf("missing repair hints: %s", output.String())
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 || entries[0].Name() != ".env" {
		t.Fatal("doctor changed disk state")
	}
}

func TestDoctorStillChecksHostWhenConfigIsInvalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	var output bytes.Buffer
	calls := 0
	err := runDoctor(path, &output, doctorChecks{
		goos: "linux", euid: 0,
		lookPath: func(name string) (string, error) { calls++; return "/usr/bin/" + name, nil },
		docker:   func(context.Context) error { calls++; return nil },
		disk:     func(context.Context, string) error { t.Fatal("disk probe used invalid config"); return nil },
	})
	if err == nil || calls != 3 || !strings.Contains(output.String(), "[OK] Docker") {
		t.Fatalf("independent checks were skipped: calls=%d error=%v output=%s", calls, err, output.String())
	}
}

func TestDoctorWarningsDoNotFailAndDisabledDiskServiceIsSkipped(t *testing.T) {
	var output bytes.Buffer
	err := runDoctor(diagnosticConfig(t, false), &output, doctorChecks{
		goos: "linux", euid: 1000,
		lookPath: func(name string) (string, error) { return "/usr/bin/" + name, nil },
		docker:   func(context.Context) error { return nil },
		disk:     func(context.Context, string) error { t.Fatal("disabled disk service probed"); return nil },
	})
	if err != nil || !strings.Contains(output.String(), "0 failure(s)") || !strings.Contains(output.String(), "[WARN]") {
		t.Fatalf("warnings treated as errors: %v; %s", err, output.String())
	}
}

func TestConfigCheckRejectsNonregularConfiguration(t *testing.T) {
	root := t.TempDir()
	pipe := filepath.Join(root, "config.pipe")
	if err := syscall.Mkfifo(pipe, 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{root, pipe} {
		if err := CheckConfig(path, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "regular file") {
			t.Fatalf("did not reject nonregular configuration: %v", err)
		}
	}
}
