package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/elecbug/linuxus/src/internal/manager/config"
)

func TestNetworkSlotsRespectAllDockerAddressRanges(t *testing.T) {
	for _, tc := range []struct {
		name    string
		subnets []string
		want    int
		fail    bool
	}{
		{"unrelated broader network", []string{"10.10.0.0/24"}, 16, false},
		{"later IPAM range", []string{"2001:db8::/64", "10.10.0.0/28"}, 1, false},
		{"narrower overlapping range", []string{"10.10.0.8/29"}, 1, false},
		{"adjacent nonoverlapping network", []string{"10.11.0.0/16"}, 0, false},
		{"all slots occupied", []string{"10.10.0.0/16"}, 0, true},
		{"invalid returned range", []string{"invalid"}, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.TrimPrefix(r.URL.Path, "/v1.47") != "/networks" || r.URL.Query().Get("filters") != "" {
					t.Errorf("network discovery was restricted: %s", r.URL)
				}
				var ranges []network.IPAMConfig
				for _, sn := range tc.subnets {
					ranges = append(ranges, network.IPAMConfig{Subnet: sn})
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode([]map[string]any{{"Name": "another-project", "Id": "external", "IPAM": network.IPAM{Config: ranges}}})
			}))
			defer backend.Close()
			cli, err := client.NewClientWithOpts(client.WithHost("tcp://"+strings.TrimPrefix(backend.URL, "http://")), client.WithVersion("1.47"), client.WithHTTPClient(backend.Client()))
			if err != nil {
				t.Fatal(err)
			}
			defer cli.Close()
			s := &Server{docker: cli, cfg: &config.Config{BaseIP: "10.10.0.0", NetworkPrefix: "net_"}}
			index, _, err := s.findFirstFreeNetworkSlot(context.Background())
			if (err != nil) != tc.fail || (!tc.fail && index != tc.want) {
				t.Fatalf("slot=%d error=%v, want slot=%d fail=%t", index, err, tc.want, tc.fail)
			}
		})
	}
}
