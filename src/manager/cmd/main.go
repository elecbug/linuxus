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
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	dockernetwork "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

const defaultAddr = ":5959"

var reInvalid = regexp.MustCompile(`[^a-z0-9]+`)

type Server struct {
	docker *client.Client
}

type CreateUserRuntimeRequest struct {
	// 필수
	UserID        string `json:"user_id"`
	Image         string `json:"image"`
	ContainerName string `json:"container_name"`

	// 네트워크 생성 규칙
	NetworkPrefix string `json:"network_prefix"` // 예: "linuxus-net-"
	BaseIP        string `json:"base_ip"`        // 예: "172.30.0.0"

	// 선택
	Hostname    string            `json:"hostname,omitempty"`
	User        string            `json:"user,omitempty"`
	WorkingDir  string            `json:"working_dir,omitempty"`
	Env         []string          `json:"env,omitempty"`
	Cmd         []string          `json:"cmd,omitempty"`
	Entrypoint  []string          `json:"entrypoint,omitempty"`
	Binds       []string          `json:"binds,omitempty"`
	Tmpfs       []string          `json:"tmpfs,omitempty"` // "/tmp:rw,noexec,nosuid,nodev,size=64m"
	CapDrop     []string          `json:"cap_drop,omitempty"`
	SecurityOpt []string          `json:"security_opt,omitempty"`
	ReadOnly    bool              `json:"read_only,omitempty"`
	Restart     string            `json:"restart,omitempty"` // "unless-stopped" 등
	Ports       []string          `json:"ports,omitempty"`   // ["10080:8080", "10022:22"]
	Labels      map[string]string `json:"labels,omitempty"`

	Memory string `json:"memory,omitempty"` // "512m"
	CPUs   string `json:"cpus,omitempty"`   // "1.5"
	Pids   int    `json:"pids,omitempty"`

	NofileSoft int `json:"nofile_soft,omitempty"`
	NofileHard int `json:"nofile_hard,omitempty"`
}

type CreateUserRuntimeResponse struct {
	OK            bool   `json:"ok"`
	UserID        string `json:"user_id,omitempty"`
	SafeID        string `json:"safe_id,omitempty"`
	ContainerName string `json:"container_name,omitempty"`
	NetworkName   string `json:"network_name,omitempty"`
	Subnet        string `json:"subnet,omitempty"`
	Index         int    `json:"index,omitempty"`
	Message       string `json:"message"`
}

func main() {
	addr := envOrDefault("LISTEN_ADDR", defaultAddr)

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("failed to create docker client: %v", err)
	}
	defer cli.Close()

	s := &Server{docker: cli}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/user/up", s.handleUserUp)

	srv := &http.Server{
		Addr:              addr,
		Handler:           loggingMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	waitForShutdown(srv)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"ok":      false,
			"message": "method not allowed",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "ready",
	})
}

func (s *Server) handleUserUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, CreateUserRuntimeResponse{
			OK:      false,
			Message: "method not allowed",
		})
		return
	}

	var req CreateUserRuntimeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, CreateUserRuntimeResponse{
			OK:      false,
			Message: "invalid json body",
		})
		return
	}

	if err := validateRequest(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, CreateUserRuntimeResponse{
			OK:      false,
			Message: err.Error(),
		})
		return
	}

	safeID := sanitizeName(req.UserID)
	if req.Restart == "" {
		req.Restart = "unless-stopped"
	}

	result, err := s.prepareRuntime(r.Context(), &req, safeID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, CreateUserRuntimeResponse{
			OK:            false,
			UserID:        req.UserID,
			SafeID:        safeID,
			ContainerName: req.ContainerName,
			Message:       err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func validateRequest(req *CreateUserRuntimeRequest) error {
	if strings.TrimSpace(req.UserID) == "" {
		return fmt.Errorf("user_id is required")
	}
	if strings.TrimSpace(req.Image) == "" {
		return fmt.Errorf("image is required")
	}
	if strings.TrimSpace(req.ContainerName) == "" {
		return fmt.Errorf("container_name is required")
	}
	if strings.TrimSpace(req.NetworkPrefix) == "" {
		return fmt.Errorf("network_prefix is required")
	}
	if strings.TrimSpace(req.BaseIP) == "" {
		return fmt.Errorf("base_ip is required")
	}
	if _, err := parseBaseIPv4(req.BaseIP); err != nil {
		return fmt.Errorf("invalid base_ip: %w", err)
	}
	return nil
}

func (s *Server) prepareRuntime(ctx context.Context, req *CreateUserRuntimeRequest, safeID string) (*CreateUserRuntimeResponse, error) {
	// 1) 이미지 존재 확인
	if err := s.ensureImageExists(ctx, req.Image); err != nil {
		return nil, err
	}

	// 2) 이미 컨테이너가 있으면 재사용
	exists, running, err := s.inspectContainerState(ctx, req.ContainerName)
	if err != nil {
		return nil, err
	}
	if exists {
		netName, subnet, idx, _ := s.findAttachedManagedNetwork(ctx, req.ContainerName, req.NetworkPrefix, req.BaseIP)
		if !running {
			if err := s.docker.ContainerStart(ctx, req.ContainerName, container.StartOptions{}); err != nil {
				return nil, fmt.Errorf("failed to start existing container: %w", err)
			}
		}
		return &CreateUserRuntimeResponse{
			OK:            true,
			UserID:        req.UserID,
			SafeID:        safeID,
			ContainerName: req.ContainerName,
			NetworkName:   netName,
			Subnet:        subnet,
			Index:         idx,
			Message:       "container already existed and is now ready",
		}, nil
	}

	// 3) 빈 네트워크 슬롯 탐색
	index, subnet, err := s.findFirstFreeNetworkSlot(ctx, req.NetworkPrefix, req.BaseIP)
	if err != nil {
		return nil, err
	}

	networkName := req.NetworkPrefix + safeID

	// safeID 기반 이름이 이미 다른 네트워크에 점유되어 있으면 충돌 방지
	networkName, err = s.ensureUniqueNetworkName(ctx, networkName, req.NetworkPrefix, index)
	if err != nil {
		return nil, err
	}

	// 4) 네트워크 생성
	if err := s.ensureNetworkWithSubnet(ctx, networkName, subnet); err != nil {
		return nil, err
	}

	// 5) 컨테이너 생성 및 시작
	if err := s.createAndStartContainer(ctx, req, networkName); err != nil {
		return nil, err
	}

	return &CreateUserRuntimeResponse{
		OK:            true,
		UserID:        req.UserID,
		SafeID:        safeID,
		ContainerName: req.ContainerName,
		NetworkName:   networkName,
		Subnet:        subnet,
		Index:         index,
		Message:       "network and container created successfully",
	}, nil
}

func (s *Server) ensureImageExists(ctx context.Context, img string) error {
	images, err := s.docker.ImageList(ctx, image.ListOptions{})
	if err == nil && len(images) == 0 {
		// no-op
	}

	_, _, err = s.docker.ImageInspectWithRaw(ctx, img)
	if err != nil {
		return fmt.Errorf("image not found or not inspectable: %s: %w", img, err)
	}
	return nil
}

func (s *Server) inspectContainerState(ctx context.Context, name string) (exists bool, running bool, err error) {
	inspect, err := s.docker.ContainerInspect(ctx, name)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, false, nil
		}
		return false, false, fmt.Errorf("failed to inspect container %s: %w", name, err)
	}
	if inspect.State != nil && inspect.State.Running {
		return true, true, nil
	}
	return true, false, nil
}

func (s *Server) findAttachedManagedNetwork(ctx context.Context, containerName, networkPrefix, baseIP string) (string, string, int, bool) {
	inspect, err := s.docker.ContainerInspect(ctx, containerName)
	if err != nil || inspect.NetworkSettings == nil {
		return "", "", -1, false
	}

	for netName := range inspect.NetworkSettings.Networks {
		if !strings.HasPrefix(netName, networkPrefix) {
			continue
		}
		netInfo, err := s.docker.NetworkInspect(ctx, netName, dockernetwork.InspectOptions{})
		if err != nil || netInfo.IPAM.Config == nil || len(netInfo.IPAM.Config) == 0 {
			continue
		}
		subnet := strings.TrimSpace(netInfo.IPAM.Config[0].Subnet)
		idx, ok := subnetToIndex(baseIP, subnet)
		if ok {
			return netName, subnet, idx, true
		}
		return netName, subnet, -1, true
	}

	return "", "", -1, false
}

func (s *Server) findFirstFreeNetworkSlot(ctx context.Context, networkPrefix, baseIP string) (int, string, error) {
	networks, err := s.docker.NetworkList(ctx, dockernetwork.ListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{
			Key:   "name",
			Value: "^" + networkPrefix,
		}),
	})
	if err != nil {
		return 0, "", fmt.Errorf("failed to list docker networks: %w", err)
	}

	used := make(map[int]struct{})

	for _, nw := range networks {
		if !strings.HasPrefix(nw.Name, networkPrefix) {
			continue
		}

		inspect, err := s.docker.NetworkInspect(ctx, nw.ID, dockernetwork.InspectOptions{})
		if err != nil {
			return 0, "", fmt.Errorf("failed to inspect network %s: %w", nw.Name, err)
		}
		if len(inspect.IPAM.Config) == 0 {
			continue
		}

		subnet := strings.TrimSpace(inspect.IPAM.Config[0].Subnet)
		idx, ok := subnetToIndex(baseIP, subnet)
		if !ok {
			continue
		}
		used[idx] = struct{}{}
	}

	index := 0
	for {
		if _, exists := used[index]; !exists {
			subnet, err := getSubnetByIndex(baseIP, index)
			if err != nil {
				return 0, "", err
			}
			return index, subnet, nil
		}
		index++
	}
}

func (s *Server) ensureUniqueNetworkName(ctx context.Context, preferredName, networkPrefix string, index int) (string, error) {
	exists, err := s.networkExists(ctx, preferredName)
	if err != nil {
		return "", err
	}
	if !exists {
		return preferredName, nil
	}

	altName := fmt.Sprintf("%sidx_%d", networkPrefix, index)
	exists, err = s.networkExists(ctx, altName)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("both preferred and fallback network names already exist: %s, %s", preferredName, altName)
	}
	return altName, nil
}

func (s *Server) networkExists(ctx context.Context, name string) (bool, error) {
	nws, err := s.docker.NetworkList(ctx, dockernetwork.ListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{
			Key:   "name",
			Value: "^" + name + "$",
		}),
	})
	if err != nil {
		return false, fmt.Errorf("failed to query network %s: %w", name, err)
	}
	for _, nw := range nws {
		if nw.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func (s *Server) ensureNetworkWithSubnet(ctx context.Context, name, subnet string) error {
	exists, err := s.networkExists(ctx, name)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	_, err = s.docker.NetworkCreate(ctx, name, dockernetwork.CreateOptions{
		Driver: "bridge",
		IPAM: &dockernetwork.IPAM{
			Config: []dockernetwork.IPAMConfig{
				{Subnet: subnet},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create network %s (%s): %w", name, subnet, err)
	}
	return nil
}

func (s *Server) createAndStartContainer(ctx context.Context, req *CreateUserRuntimeRequest, networkName string) error {
	cfg := &container.Config{
		Image:      req.Image,
		Hostname:   req.Hostname,
		User:       req.User,
		WorkingDir: req.WorkingDir,
		Env:        req.Env,
		Cmd:        req.Cmd,
		Entrypoint: req.Entrypoint,
		Labels:     req.Labels,
	}

	exposedPorts, portBindings, err := parsePortBindings(req.Ports)
	if err != nil {
		return err
	}
	cfg.ExposedPorts = exposedPorts

	hostCfg := &container.HostConfig{
		Binds:          req.Binds,
		Tmpfs:          parseSliceToTmpfsMap(req.Tmpfs),
		ReadonlyRootfs: req.ReadOnly,
		SecurityOpt:    req.SecurityOpt,
		CapDrop:        req.CapDrop,
		PortBindings:   portBindings,
		RestartPolicy: container.RestartPolicy{
			Name: container.RestartPolicyMode(req.Restart),
		},
	}

	if req.Memory != "" {
		memBytes, err := parseMemoryBytes(req.Memory)
		if err != nil {
			return fmt.Errorf("invalid memory limit %q: %w", req.Memory, err)
		}
		hostCfg.Memory = memBytes
	}

	if req.CPUs != "" {
		nanoCPUs, err := parseNanoCPUs(req.CPUs)
		if err != nil {
			return fmt.Errorf("invalid cpu limit %q: %w", req.CPUs, err)
		}
		hostCfg.NanoCPUs = nanoCPUs
	}

	if req.Pids > 0 {
		pidsLimit := int64(req.Pids)
		hostCfg.PidsLimit = &pidsLimit
	}

	if req.NofileSoft > 0 || req.NofileHard > 0 {
		hostCfg.Ulimits = append(hostCfg.Ulimits, &container.Ulimit{
			Name: "nofile",
			Soft: int64(req.NofileSoft),
			Hard: int64(req.NofileHard),
		})
	}

	networkingCfg := &dockernetwork.NetworkingConfig{
		EndpointsConfig: map[string]*dockernetwork.EndpointSettings{
			networkName: {},
		},
	}

	resp, err := s.docker.ContainerCreate(ctx, cfg, hostCfg, networkingCfg, nil, req.ContainerName)
	if err != nil {
		return fmt.Errorf("failed to create container %s: %w", req.ContainerName, err)
	}

	if err := s.docker.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container %s: %w", req.ContainerName, err)
	}

	return nil
}

func parseBaseIPv4(s string) (net.IP, error) {
	ip := net.ParseIP(strings.TrimSpace(s))
	if ip == nil {
		return nil, fmt.Errorf("not an ip")
	}
	ip = ip.To4()
	if ip == nil {
		return nil, fmt.Errorf("not an ipv4")
	}
	return ip, nil
}

// 규칙:
// index 0 -> baseIP의 3옥텟 유지, 4옥텟 0
// index 1 -> 4옥텟 16
// ...
// index 15 -> 4옥텟 240
// index 16 -> 3옥텟 +1, 4옥텟 0
func getSubnetByIndex(baseIP string, index int) (string, error) {
	ip, err := parseBaseIPv4(baseIP)
	if err != nil {
		return "", err
	}
	if index < 0 {
		return "", fmt.Errorf("index must be non-negative")
	}

	o0, o1, o2 := int(ip[0]), int(ip[1]), int(ip[2])

	thirdOffset := index / 16
	fourthOffset := (index % 16) * 16

	newO2 := o2 + thirdOffset
	if newO2 > 255 {
		return "", fmt.Errorf("subnet overflow: 3rd octet > 255")
	}

	return fmt.Sprintf("%d.%d.%d.%d/28", o0, o1, newO2, fourthOffset), nil
}

func subnetToIndex(baseIP, subnet string) (int, bool) {
	base, err := parseBaseIPv4(baseIP)
	if err != nil {
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

func parsePortBindings(items []string) (nat.PortSet, nat.PortMap, error) {
	if len(items) == 0 {
		return nil, nil, nil
	}

	exposed := nat.PortSet{}
	bindings := nat.PortMap{}

	for _, item := range items {
		containerPort, hostBinding, err := parsePortBinding(item)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid port binding %q: %w", item, err)
		}
		exposed[containerPort] = struct{}{}
		bindings[containerPort] = append(bindings[containerPort], hostBinding)
	}

	return exposed, bindings, nil
}

func parsePortBinding(s string) (nat.Port, nat.PortBinding, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return "", nat.PortBinding{}, fmt.Errorf("expected HOST:CONTAINER, got %q", s)
	}

	hostPort := strings.TrimSpace(parts[0])
	containerPart := strings.TrimSpace(parts[1])

	containerPort, err := nat.NewPort("tcp", containerPart)
	if err != nil {
		return "", nat.PortBinding{}, err
	}

	return containerPort, nat.PortBinding{
		HostIP:   "",
		HostPort: hostPort,
	}, nil
}

func parseSliceToTmpfsMap(items []string) map[string]string {
	if len(items) == 0 {
		return nil
	}

	out := make(map[string]string, len(items))
	for _, item := range items {
		parts := strings.SplitN(item, ":", 2)
		mountPoint := strings.TrimSpace(parts[0])
		if mountPoint == "" {
			continue
		}

		opts := ""
		if len(parts) == 2 {
			opts = strings.TrimSpace(parts[1])
		}
		out[mountPoint] = opts
	}
	return out
}

func parseNanoCPUs(v string) (int64, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return 0, err
	}
	if f < 0 {
		return 0, fmt.Errorf("must be non-negative")
	}
	return int64(f * 1_000_000_000), nil
}

func parseMemoryBytes(v string) (int64, error) {
	s := strings.TrimSpace(strings.ToLower(v))
	mult := int64(1)

	switch {
	case strings.HasSuffix(s, "g"):
		mult = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "g")
	case strings.HasSuffix(s, "gb"):
		mult = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "gb")
	case strings.HasSuffix(s, "m"):
		mult = 1024 * 1024
		s = strings.TrimSuffix(s, "m")
	case strings.HasSuffix(s, "mb"):
		mult = 1024 * 1024
		s = strings.TrimSuffix(s, "mb")
	case strings.HasSuffix(s, "k"):
		mult = 1024
		s = strings.TrimSuffix(s, "k")
	case strings.HasSuffix(s, "kb"):
		mult = 1024
		s = strings.TrimSuffix(s, "kb")
	case strings.HasSuffix(s, "b"):
		mult = 1
		s = strings.TrimSuffix(s, "b")
	}

	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, fmt.Errorf("must be non-negative")
	}
	return n * mult, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func envOrDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func waitForShutdown(srv *http.Server) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	log.Printf("received signal: %s", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		_ = srv.Close()
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s from=%s took=%s", r.Method, r.URL.String(), r.RemoteAddr, time.Since(start))
	})
}
