package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

func TestNetworkSubnetsHandlesEmptyAndDualStackRanges(t *testing.T) {
	for _, tc := range []struct {
		ranges []network.IPAMConfig
		want   string
	}{
		{nil, "-"},
		{[]network.IPAMConfig{{}}, "-"},
		{[]network.IPAMConfig{{Subnet: "172.20.0.0/24"}, {}, {Subnet: "fd00::/64"}}, "172.20.0.0/24, fd00::/64"},
	} {
		if got := networkSubnets(tc.ranges); got != tc.want {
			t.Fatalf("subnets=%q, want %q", got, tc.want)
		}
	}
}

func TestStatusHandlesPartialDockerInspection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch path := strings.TrimPrefix(r.URL.Path, "/v1.47"); path {
		case "/containers/json":
			w.Write([]byte(`[]`))
		case "/containers/auth/json", "/containers/manager/json":
			w.Write([]byte(`{"Id":"test","Config":null,"State":null}`))
		case "/networks":
			w.Write([]byte(`[{"Name":"manager-net","Id":"abc","IPAM":{"Config":[]}}]`))
		default:
			t.Errorf("unexpected API: %s", path)
			w.WriteHeader(500)
		}
	}))
	defer server.Close()
	cli, err := client.NewClientWithOpts(client.WithHost("tcp://"+strings.TrimPrefix(server.URL, "http://")), client.WithVersion("1.47"), client.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()
	a := &App{dockerClient: cli, context: context.Background()}
	a.Config.AuthService.Container.Name = "auth"
	a.Config.ManagerService.Container.Name = "manager"
	a.Config.ManagerService.Container.Network = "manager-net"
	a.Config.UserService.Container.NamePrefix = "user-"
	a.Config.UserService.Container.NetworkNamePrefix = "user-net-"
	if err := a.showContainerInfos(); err != nil {
		t.Fatal(err)
	}
	if err := a.showNetworkInfos(); err != nil {
		t.Fatal(err)
	}
}
