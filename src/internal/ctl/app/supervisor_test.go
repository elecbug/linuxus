package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/docker/docker/client"
	"path/filepath"
	"strings"
	"testing"
)

func TestSystemdUnitPinsPathsAndOwnsChildren(t *testing.T) {
	var output bytes.Buffer
	if err := WriteSystemdUnit("/opt/class %1/$ctl", "/etc/classroom settings/.env", &output); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	for _, want := range []string{"Requires=docker.service", "After=docker.service local-fs.target", "KillMode=control-group", "Restart=on-failure", "class %%1/$$ctl", "LINUXUS_CONFIG=/etc/classroom settings/.env", "supervise"} {
		if !strings.Contains(text, want) {
			t.Errorf("unit missing %q", want)
		}
	}
	if err := WriteSystemdUnit("/tmp/ctl\nExecStart=/bad", "/etc/.env", &bytes.Buffer{}); err == nil {
		t.Fatal("accepted unit directive injection")
	}
	a := &App{supervised: true}
	a.Config.AuthService.Mounts.HostAuthListPath = filepath.Join(t.TempDir(), "AUTH_LIST")
	if a.serviceRestartPolicy() != "no" {
		t.Fatal("Docker may start supervised services before mounts")
	}
}

type recoveryTransport func(*http.Request) (*http.Response, error)

func (f recoveryTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSupervisorRecoveryBoundsCreateAndHonorsCancellation(t *testing.T) {
	for _, service := range []string{"auth", "manager"} {
		t.Run(service, func(t *testing.T) {
			entered := make(chan bool, 1)
			transport := recoveryTransport(func(r *http.Request) (*http.Response, error) {
				path := strings.TrimPrefix(r.URL.Path, "/v1.47")
				if path == "/containers/create" {
					deadline, ok := r.Context().Deadline()
					entered <- ok && time.Until(deadline) > 0 && time.Until(deadline) <= time.Minute
					<-r.Context().Done()
					return nil, r.Context().Err()
				}
				if path != "/containers/json" {
					return nil, fmt.Errorf("unexpected Docker API: %s", path)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[]`)), Header: make(http.Header)}, nil
			})
			cli, err := client.NewClientWithOpts(client.WithHost("tcp://127.0.0.1:2375"), client.WithVersion("1.47"), client.WithHTTPClient(&http.Client{Transport: transport}))
			if err != nil {
				t.Fatal(err)
			}
			defer cli.Close()
			a := &App{dockerClient: cli, context: context.Background()}
			a.Config.AuthService.Container.Name = "auth"
			a.Config.AuthService.Container.ExternalPort = 8080
			a.Config.ManagerService.Container.Name = "manager"
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- a.recoverServiceContainer(ctx, service) }()
			select {
			case bounded := <-entered:
				cancel()
				if !bounded {
					t.Fatal("Docker create did not receive a bounded recovery deadline")
				}
			case err := <-done:
				t.Fatalf("recovery failed before create: %v", err)
			case <-time.After(2 * time.Second):
				t.Fatal("recovery did not reach create")
			}
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("recovery lost cancellation: %v", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("Docker recovery ignored supervisor shutdown")
			}
			if a.context.Err() != nil {
				t.Fatal("recovery changed the supervisor's context")
			}
		})
	}
}
