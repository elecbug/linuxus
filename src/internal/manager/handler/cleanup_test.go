package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/docker/docker/client"
	"github.com/elecbug/linuxus/src/internal/manager/config"
)

func TestRuntimeCleanupOnlyRemovesIdentifiedNetworks(t *testing.T) {
	for _, mode := range []string{"inspect-failure", "missing-unowned", "missing-owned", "existing", "no-managed-network", "retry-network-failure", "orphan-list-failure"} {
		t.Run(mode, func(t *testing.T) {
			mutations := make(chan string, 16)
			var removed atomic.Bool
			var networkRemovals atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := strings.TrimPrefix(r.URL.Path, "/v1.47")
				w.Header().Set("Content-Type", "application/json")
				if r.Method != http.MethodGet {
					mutations <- r.Method + " " + path
					if r.Method == http.MethodDelete && strings.HasPrefix(path, "/containers/") {
						removed.Store(true)
					}
					if mode == "retry-network-failure" && r.Method == http.MethodDelete && path == "/networks/owned-id" && networkRemovals.Add(1) == 1 {
						w.WriteHeader(http.StatusConflict)
						json.NewEncoder(w).Encode(map[string]string{"message": "network is busy"})
						return
					}
					w.WriteHeader(http.StatusNoContent)
					return
				}
				switch path {
				case "/containers/user_alice/json":
					if mode == "inspect-failure" || strings.HasPrefix(mode, "missing-") || mode == "orphan-list-failure" || removed.Load() {
						status := http.StatusNotFound
						if mode == "inspect-failure" {
							status = http.StatusInternalServerError
						}
						w.WriteHeader(status)
						json.NewEncoder(w).Encode(map[string]string{"message": "cannot inspect container"})
						return
					}
					networks := map[string]any{"net_idx_3": map[string]string{"NetworkID": "owned-id"}}
					if mode == "no-managed-network" {
						networks = map[string]any{"unrelated": map[string]string{"NetworkID": "unrelated-id"}}
					}
					json.NewEncoder(w).Encode(map[string]any{"Id": "container-id", "NetworkSettings": map[string]any{"Networks": networks}})
				case "/networks":
					if mode == "orphan-list-failure" {
						w.WriteHeader(http.StatusInternalServerError)
						json.NewEncoder(w).Encode(map[string]string{"message": "cannot list networks"})
						return
					}
					rows := []map[string]any{{"Name": "net_alice", "Id": "unrelated-id", "Labels": map[string]string{}}}
					if mode == "missing-owned" || mode == "retry-network-failure" {
						rows = append(rows, map[string]any{"Name": "net_idx_3", "Id": "owned-id", "Labels": map[string]string{"io.linuxus.user-container": "user_alice"}})
					}
					json.NewEncoder(w).Encode(rows)
				default:
					t.Errorf("unexpected inspection: %s", path)
					http.Error(w, "unexpected API", 500)
				}
			}))
			defer server.Close()
			cli, err := client.NewClientWithOpts(client.WithHost("tcp://"+strings.TrimPrefix(server.URL, "http://")), client.WithVersion("1.47"), client.WithHTTPClient(server.Client()))
			if err != nil {
				t.Fatal(err)
			}
			defer cli.Close()
			s := &Server{docker: cli, cfg: &config.Config{UserContainerNamePrefix: "user_", NetworkPrefix: "net_", AuthContainerName: "auth"}}
			err = s.stopAndRemoveUserContainerAndNetwork(context.Background(), "alice")
			wantError := mode == "inspect-failure" || mode == "retry-network-failure" || mode == "orphan-list-failure"
			if (err != nil) != wantError {
				t.Fatalf("cleanup error: %v", err)
			}
			if mode == "retry-network-failure" {
				if err := s.stopAndRemoveUserContainerAndNetwork(context.Background(), "alice"); err != nil {
					t.Fatalf("cleanup did not recover after container removal: %v", err)
				}
				if networkRemovals.Load() != 2 {
					t.Fatal("cleanup failed to retry the owned orphan network")
				}
			}
			var calls []string
			for len(mutations) > 0 {
				calls = append(calls, <-mutations)
			}
			if (mode == "inspect-failure" || mode == "orphan-list-failure") && len(calls) != 0 {
				t.Fatalf("cleanup changed resources after failed inspection: %v", calls)
			}
			wantNetwork := mode == "existing" || mode == "missing-owned" || mode == "retry-network-failure"
			if slices.Contains(calls, "DELETE /networks/owned-id") != wantNetwork {
				t.Fatalf("owned network was not cleaned by ID: %v", calls)
			}
			for _, call := range calls {
				if strings.Contains(call, "/networks/") && call != "DELETE /networks/owned-id" && call != "POST /networks/owned-id/disconnect" {
					t.Fatalf("cleanup selected a network without ownership evidence: %s", call)
				}
			}
		})
	}
}
