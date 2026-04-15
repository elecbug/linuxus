package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

var reInvalid = regexp.MustCompile(`[^a-z0-9]+`)

type Server struct {
	docker *client.Client
	cfg    Config
}

type Config struct {
	ListenAddr              string
	UserImage               string
	UserContainerNamePrefix string
	NetworkPrefix           string
	BaseIP                  string
	AuthContainerName       string
	AdminUserID             string

	RuntimeUser          string
	ContainerRuntimeUser string
	ContainerHostname    string
	WorkingDir           string
	Timezone             string
	ReadOnlyRootFS       bool
	ManagerWaitTime      time.Duration

	HostHomesDir         string
	HostShareDir         string
	HostReadonlyDir      string
	ContainerShareDir    string
	ContainerReadonlyDir string
}

type UserUpRequest struct {
	UserID string `json:"user_id"`
	SafeID string `json:"safe_id"`
}

type UserUpResponse struct {
	OK            bool   `json:"ok"`
	UserID        string `json:"user_id,omitempty"`
	SafeID        string `json:"safe_id,omitempty"`
	ContainerName string `json:"container_name,omitempty"`
	NetworkName   string `json:"network_name,omitempty"`
	Subnet        string `json:"subnet,omitempty"`
	Message       string `json:"message"`
}

func main() {
	waitTime := 10 * time.Second
	if raw := envOr("MANAGER_WAIT_TIME", "10s"); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil {
			waitTime = d
		}
	}

	cfg := Config{
		ListenAddr:              envOr("LISTEN_ADDR", ":5959"),
		UserImage:               envOr("USER_IMAGE", "linuxus-user-base:runtime"),
		UserContainerNamePrefix: envOr("USER_CONTAINER_NAME_PREFIX", "linuxus-user-"),
		NetworkPrefix:           envOr("NETWORK_PREFIX", "linuxus-net-"),
		BaseIP:                  envOr("BASE_IP", "172.30.0.0"),
		AuthContainerName:       envOr("AUTH_CONTAINER_NAME", "linuxus-auth"),
		AdminUserID:             envOr("ADMIN_USER_ID", "alpha"),

		RuntimeUser:          envOr("RUNTIME_USER", "1000:1000"),
		ContainerRuntimeUser: envOr("CONTAINER_RUNTIME_USER", "student"),
		ContainerHostname:    envOr("CONTAINER_HOSTNAME", "linuxus"),
		WorkingDir:           envOr("WORKING_DIR", "/home/student"),
		Timezone:             envOr("TZ", "Asia/Seoul"),
		ReadOnlyRootFS:       true,
		ManagerWaitTime:      waitTime,

		HostHomesDir:         envOr("HOST_HOMES_DIR", ""),
		HostShareDir:         envOr("HOST_SHARE_DIR", ""),
		HostReadonlyDir:      envOr("HOST_READONLY_DIR", ""),
		ContainerShareDir:    envOr("CONTAINER_SHARE_DIR", "/share"),
		ContainerReadonlyDir: envOr("CONTAINER_READONLY_DIR", "/readonly"),
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("docker client init failed: %v", err)
	}
	defer cli.Close()

	s := &Server{docker: cli, cfg: cfg}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/user/up", s.handleUserUp)

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("manager listening on %s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen failed: %v", err)
		}
	}()

	waitForShutdown(srv)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true,
	})
}

func (s *Server) handleUserUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, UserUpResponse{
			OK:      false,
			Message: "method not allowed",
		})
		return
	}

	var req UserUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, UserUpResponse{
			OK:      false,
			Message: "invalid json body",
		})
		return
	}

	req.UserID = strings.TrimSpace(req.UserID)
	req.SafeID = strings.TrimSpace(req.SafeID)
	if req.UserID == "" {
		writeJSON(w, http.StatusBadRequest, UserUpResponse{
			OK:      false,
			Message: "user_id is required",
		})
		return
	}
	if req.SafeID == "" {
		req.SafeID = sanitizeName(req.UserID)
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.ManagerWaitTime)
	defer cancel()

	resp, err := s.ensureUserRuntimeReady(ctx, req.UserID, req.SafeID)
	if err != nil {
		log.Printf("user up failed user=%s safe=%s err=%v", req.UserID, req.SafeID, err)
		writeJSON(w, http.StatusServiceUnavailable, UserUpResponse{
			OK:            false,
			UserID:        req.UserID,
			SafeID:        req.SafeID,
			ContainerName: s.cfg.UserContainerNamePrefix + req.SafeID,
			Message:       err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) ensureUserRuntimeReady(ctx context.Context, userID, safeID string) (*UserUpResponse, error) {
	containerName := s.cfg.UserContainerNamePrefix + safeID

	if _, _, err := s.docker.ImageInspectWithRaw(ctx, s.cfg.UserImage); err != nil {
		return nil, fmt.Errorf("user image not found: %s", s.cfg.UserImage)
	}

	exists, running, err := s.inspectContainerState(ctx, containerName)
	if err != nil {
		return nil, err
	}

	if exists {
		networkName, subnet, err := s.ensureExistingContainerNetworkAndAuth(ctx, containerName)
		if err != nil {
			return nil, err
		}

		if !running {
			if err := s.docker.ContainerStart(ctx, containerName, container.StartOptions{}); err != nil {
				return nil, fmt.Errorf("failed to start existing container: %w", err)
			}
		}

		if _, err := s.waitForContainerIP(ctx, containerName, networkName); err != nil {
			return nil, err
		}

		return &UserUpResponse{
			OK:            true,
			UserID:        userID,
			SafeID:        safeID,
			ContainerName: containerName,
			NetworkName:   networkName,
			Subnet:        subnet,
			Message:       "container ready",
		}, nil
	}

	index, subnet, err := s.findFirstFreeNetworkSlot(ctx)
	if err != nil {
		return nil, err
	}

	networkName := s.cfg.NetworkPrefix + safeID
	if exists, err := s.networkExists(ctx, networkName); err != nil {
		return nil, err
	} else if exists {
		networkName = fmt.Sprintf("%sidx_%d", s.cfg.NetworkPrefix, index)
	}

	if err := s.createNetwork(ctx, networkName, subnet); err != nil {
		return nil, err
	}

	if err := s.createUserContainer(ctx, containerName, userID, safeID, networkName); err != nil {
		return nil, err
	}

	if err := s.ensureAuthConnected(ctx, networkName); err != nil {
		return nil, err
	}

	if _, err := s.waitForContainerIP(ctx, containerName, networkName); err != nil {
		return nil, err
	}

	return &UserUpResponse{
		OK:            true,
		UserID:        userID,
		SafeID:        safeID,
		ContainerName: containerName,
		NetworkName:   networkName,
		Subnet:        subnet,
		Message:       "container ready",
	}, nil
}

func (s *Server) inspectContainerState(ctx context.Context, name string) (bool, bool, error) {
	inspect, err := s.docker.ContainerInspect(ctx, name)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, false, nil
		}
		return false, false, fmt.Errorf("container inspect failed: %w", err)
	}
	if inspect.State != nil && inspect.State.Running {
		return true, true, nil
	}
	return true, false, nil
}

func (s *Server) ensureExistingContainerNetworkAndAuth(ctx context.Context, containerName string) (string, string, error) {
	inspect, err := s.docker.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", "", fmt.Errorf("inspect existing container failed: %w", err)
	}
	if inspect.NetworkSettings == nil || len(inspect.NetworkSettings.Networks) == 0 {
		return "", "", fmt.Errorf("existing container has no network")
	}

	for netName := range inspect.NetworkSettings.Networks {
		if !strings.HasPrefix(netName, s.cfg.NetworkPrefix) {
			continue
		}
		netInfo, err := s.docker.NetworkInspect(ctx, netName, network.InspectOptions{})
		if err != nil {
			return "", "", fmt.Errorf("network inspect failed: %w", err)
		}

		subnet := ""
		if len(netInfo.IPAM.Config) > 0 {
			subnet = strings.TrimSpace(netInfo.IPAM.Config[0].Subnet)
		}

		if err := s.ensureAuthConnected(ctx, netName); err != nil {
			return "", "", err
		}
		return netName, subnet, nil
	}

	return "", "", fmt.Errorf("existing container is not attached to managed network")
}

func (s *Server) createUserContainer(ctx context.Context, containerName, userID, safeID, networkName string) error {
	homeDir := strings.TrimRight(s.cfg.HostHomesDir, "/") + "/" + userID

	cfg := &container.Config{
		Image:      s.cfg.UserImage,
		Hostname:   s.cfg.ContainerHostname,
		User:       s.cfg.RuntimeUser,
		WorkingDir: s.cfg.WorkingDir,
		Env: []string{
			"TZ=" + s.cfg.Timezone,
			"CONTAINER_RUNTIME_USER=" + s.cfg.ContainerRuntimeUser,
			"USER_ID=" + userID,
			"SHARED_DIR=" + s.cfg.ContainerShareDir,
			"READONLY_DIR=" + s.cfg.ContainerReadonlyDir,
			"IS_ADMIN=false",
		},
	}

	hostCfg := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/home/%s:rw", homeDir, s.cfg.ContainerRuntimeUser),
			fmt.Sprintf("%s:%s:rw", s.cfg.HostShareDir, s.cfg.ContainerShareDir),
			getReadonlyBind(userID, &s.cfg),
		},
		Tmpfs: map[string]string{
			"/tmp":     "rw,noexec,nosuid,nodev,size=64m",
			"/run":     "rw,noexec,nosuid,nodev,size=16m",
			"/var/tmp": "rw,noexec,nosuid,nodev,size=64m",
		},
		ReadonlyRootfs: s.cfg.ReadOnlyRootFS,
		SecurityOpt:    []string{"no-new-privileges:true"},
		CapDrop:        []string{"ALL"},
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
	}

	networkingCfg := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkName: {},
		},
	}

	resp, err := s.docker.ContainerCreate(ctx, cfg, hostCfg, networkingCfg, nil, containerName)
	if err != nil {
		return fmt.Errorf("container create failed: %w", err)
	}

	if err := s.docker.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("container start failed: %w", err)
	}

	return nil
}

func getReadonlyBind(userID string, cfg *Config) string {
	if userID == cfg.AdminUserID {
		return fmt.Sprintf("%s:%s:rw", cfg.HostReadonlyDir, cfg.ContainerReadonlyDir)
	} else {
		return fmt.Sprintf("%s:%s:ro", cfg.HostReadonlyDir, cfg.ContainerReadonlyDir)
	}
}

func (s *Server) ensureAuthConnected(ctx context.Context, networkName string) error {
	netInfo, err := s.docker.NetworkInspect(ctx, networkName, network.InspectOptions{})
	if err != nil {
		return fmt.Errorf("network inspect failed: %w", err)
	}

	for _, endpoint := range netInfo.Containers {
		if endpoint.Name == s.cfg.AuthContainerName {
			return nil
		}
	}

	if err := s.docker.NetworkConnect(ctx, networkName, s.cfg.AuthContainerName, &network.EndpointSettings{}); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "already exists") ||
			strings.Contains(strings.ToLower(err.Error()), "already connected") {
			return nil
		}
		return fmt.Errorf("failed to connect auth container to %s: %w", networkName, err)
	}

	return nil
}

func (s *Server) waitForContainerIP(ctx context.Context, containerName, networkName string) (string, error) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		ip, err := s.containerIPv4OnNetwork(ctx, containerName, networkName)
		if err == nil && strings.TrimSpace(ip) != "" {
			return ip, nil
		}

		select {
		case <-ctx.Done():
			return "", fmt.Errorf("container network was not ready within timeout")
		case <-ticker.C:
		}
	}
}

func (s *Server) containerIPv4OnNetwork(ctx context.Context, containerName, networkName string) (string, error) {
	inspect, err := s.docker.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", fmt.Errorf("container inspect failed: %w", err)
	}
	if inspect.NetworkSettings == nil {
		return "", fmt.Errorf("container has no network settings")
	}
	ep, ok := inspect.NetworkSettings.Networks[networkName]
	if !ok || ep == nil {
		return "", fmt.Errorf("container is not attached to network %s", networkName)
	}
	if strings.TrimSpace(ep.IPAddress) == "" {
		return "", fmt.Errorf("container has no ipv4 on network %s", networkName)
	}
	return ep.IPAddress, nil
}

func (s *Server) findFirstFreeNetworkSlot(ctx context.Context) (int, string, error) {
	networks, err := s.docker.NetworkList(ctx, network.ListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{
			Key:   "name",
			Value: "^" + s.cfg.NetworkPrefix,
		}),
	})
	if err != nil {
		return 0, "", fmt.Errorf("network list failed: %w", err)
	}

	used := make(map[int]struct{})

	for _, nw := range networks {
		if !strings.HasPrefix(nw.Name, s.cfg.NetworkPrefix) {
			continue
		}

		inspect, err := s.docker.NetworkInspect(ctx, nw.ID, network.InspectOptions{})
		if err != nil {
			return 0, "", fmt.Errorf("network inspect failed: %w", err)
		}
		if len(inspect.IPAM.Config) == 0 {
			continue
		}

		subnet := strings.TrimSpace(inspect.IPAM.Config[0].Subnet)
		idx, ok := subnetToIndex(s.cfg.BaseIP, subnet)
		if ok {
			used[idx] = struct{}{}
		}
	}

	for idx := 0; ; idx++ {
		if _, exists := used[idx]; exists {
			continue
		}
		subnet, err := getSubnetByIndex(s.cfg.BaseIP, idx)
		if err != nil {
			return 0, "", err
		}
		return idx, subnet, nil
	}
}

func (s *Server) networkExists(ctx context.Context, name string) (bool, error) {
	nws, err := s.docker.NetworkList(ctx, network.ListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{
			Key:   "name",
			Value: "^" + name + "$",
		}),
	})
	if err != nil {
		return false, fmt.Errorf("network exists query failed: %w", err)
	}
	for _, nw := range nws {
		if nw.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func (s *Server) createNetwork(ctx context.Context, name, subnet string) error {
	if exists, err := s.networkExists(ctx, name); err != nil {
		return err
	} else if exists {
		return nil
	}

	_, err := s.docker.NetworkCreate(ctx, name, network.CreateOptions{
		Driver: "bridge",
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{
				{Subnet: subnet},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("network create failed: %w", err)
	}
	return nil
}

func getSubnetByIndex(baseIP string, index int) (string, error) {
	ip := net.ParseIP(strings.TrimSpace(baseIP)).To4()
	if ip == nil {
		return "", fmt.Errorf("invalid base ip")
	}
	if index < 0 {
		return "", fmt.Errorf("invalid index")
	}

	o0, o1, o2 := int(ip[0]), int(ip[1]), int(ip[2])
	thirdOffset := index / 16
	fourthOffset := (index % 16) * 16

	newO2 := o2 + thirdOffset
	if newO2 > 255 {
		return "", fmt.Errorf("subnet overflow")
	}

	return fmt.Sprintf("%d.%d.%d.%d/28", o0, o1, newO2, fourthOffset), nil
}

func subnetToIndex(baseIP, subnet string) (int, bool) {
	base := net.ParseIP(strings.TrimSpace(baseIP)).To4()
	if base == nil {
		return 0, false
	}

	ip, ipNet, err := net.ParseCIDR(strings.TrimSpace(subnet))
	if err != nil {
		return 0, false
	}
	ip = ip.To4()
	if ip == nil {
		return 0, false
	}

	ones, bits := ipNet.Mask.Size()
	if bits != 32 || ones != 28 {
		return 0, false
	}

	if ip[0] != base[0] || ip[1] != base[1] {
		return 0, false
	}
	if ip[3]%16 != 0 {
		return 0, false
	}
	if ip[2] < base[2] {
		return 0, false
	}

	thirdDelta := int(ip[2]) - int(base[2])
	index := thirdDelta*16 + int(ip[3])/16
	return index, true
}

func sanitizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = reInvalid.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		return "invalid"
	}
	return s
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func envOr(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func waitForShutdown(srv *http.Server) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	_ = srv.Close()
}
