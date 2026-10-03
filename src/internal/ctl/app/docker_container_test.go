package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docker/docker/client"
	"github.com/elecbug/linuxus/src/internal/ctl/spec"
)

func TestContainerCreationValidationAndRollback(t *testing.T) {
	for _, tc := range []struct {
		name         string
		existing     bool
		invalidLimit bool
		networks     []string
		failPath     string
		wantError    bool
		wantDeletes  int
	}{
		{"no_networks", false, false, nil, "", false, 0},
		{"invalid_settings_preserve_running_container", true, true, nil, "", true, 0},
		{"start_failure", false, false, []string{"main"}, "/containers/created-id/start", true, 1},
		{"network_failure", false, false, []string{"main", "extra"}, "/networks/extra/connect", true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deletes := make(chan string, 4)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := strings.TrimPrefix(r.URL.Path, "/v1.47")
				w.Header().Set("Content-Type", "application/json")
				if path == tc.failPath {
					w.WriteHeader(500)
					w.Write([]byte(`{"message":"injected failure"}`))
					return
				}
				switch {
				case r.Method == "DELETE":
					deletes <- path
					w.WriteHeader(204)
				case path == "/containers/json":
					if tc.existing {
						w.Write([]byte(`[{"Id":"old","Names":["/auth"]}]`))
					} else {
						w.Write([]byte(`[]`))
					}
				case path == "/containers/create":
					w.WriteHeader(201)
					w.Write([]byte(`{"Id":"created-id"}`))
				case strings.HasSuffix(path, "/start") || strings.HasSuffix(path, "/connect"):
					w.WriteHeader(204)
				default:
					w.WriteHeader(500)
					json.NewEncoder(w).Encode(map[string]string{"message": "unexpected API: " + path})
				}
			}))
			defer server.Close()
			cli, err := client.NewClientWithOpts(client.WithHost("tcp://"+strings.TrimPrefix(server.URL, "http://")), client.WithVersion("1.47"), client.WithHTTPClient(server.Client()))
			if err != nil {
				t.Fatal(err)
			}
			defer cli.Close()
			a := &App{dockerClient: cli, context: context.Background()}
			spec := spec.RuntimeContainerSpec{Name: "auth", Image: "test", Networks: tc.networks}
			if tc.invalidLimit {
				spec.Limits.Memory = "invalid"
			}
			err = a.ensureContainer(spec)
			if (err != nil) != tc.wantError {
				t.Fatalf("error=%v", err)
			}
			if len(deletes) != tc.wantDeletes {
				t.Fatalf("removed %d containers, expected %d", len(deletes), tc.wantDeletes)
			}
			for len(deletes) > 0 {
				if id := <-deletes; id != "/containers/created-id" {
					t.Fatalf("removed unrelated container %s", id)
				}
			}
		})
	}
}

func TestInvalidPortPreservesExistingContainer(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		t.Errorf("invalid port reached Docker API: %s", r.URL.Path)
		w.WriteHeader(500)
	}))
	defer server.Close()
	cli, err := client.NewClientWithOpts(client.WithHost("tcp://"+strings.TrimPrefix(server.URL, "http://")), client.WithVersion("1.47"), client.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()
	a := &App{dockerClient: cli, context: context.Background()}
	for _, port := range []string{"invalid:8080", "65536:8080", "8080:0"} {
		if err := a.ensureContainer(spec.RuntimeContainerSpec{Name: "auth", Image: "test", Ports: []string{port}}); err == nil {
			t.Fatalf("accepted port %q", port)
		}
	}
	// HTTP handlers are joined by Close before the server's state is examined.
	server.Close()
	if calls != 0 {
		t.Fatal("modified Docker resources")
	}
}
