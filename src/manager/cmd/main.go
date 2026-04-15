package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/elecbug/linuxus/src/manager/internal/config"
	"github.com/elecbug/linuxus/src/manager/internal/handler"
)

func main() {
	cfg, err := parseConfigFromEnv()
	if err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	s, err := handler.NewServer(cfg)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.HandleHealthz)
	mux.HandleFunc("/user/up", s.HandleUserUp)

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

func parseConfigFromEnv() (*config.Config, error) {
	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		return nil, fmt.Errorf("LISTEN_ADDR is required")
	}
	userImage := os.Getenv("USER_IMAGE")
	if userImage == "" {
		return nil, fmt.Errorf("USER_IMAGE is required")
	}
	userContainerNamePrefix := os.Getenv("USER_CONTAINER_NAME_PREFIX")
	if userContainerNamePrefix == "" {
		return nil, fmt.Errorf("USER_CONTAINER_NAME_PREFIX is required")
	}
	networkPrefix := os.Getenv("NETWORK_PREFIX")
	if networkPrefix == "" {
		return nil, fmt.Errorf("NETWORK_PREFIX is required")
	}
	baseIP := os.Getenv("BASE_IP")
	if baseIP == "" {
		return nil, fmt.Errorf("BASE_IP is required")
	}
	authContainerName := os.Getenv("AUTH_CONTAINER_NAME")
	if authContainerName == "" {
		return nil, fmt.Errorf("AUTH_CONTAINER_NAME is required")
	}
	adminUserID := os.Getenv("ADMIN_USER_ID")
	if adminUserID == "" {
		return nil, fmt.Errorf("ADMIN_USER_ID is required")
	}

	runtimeUser := os.Getenv("RUNTIME_USER")
	if runtimeUser == "" {
		return nil, fmt.Errorf("RUNTIME_USER is required")
	}
	containerRuntimeUser := os.Getenv("CONTAINER_RUNTIME_USER")
	if containerRuntimeUser == "" {
		return nil, fmt.Errorf("CONTAINER_RUNTIME_USER is required")
	}
	containerHostname := os.Getenv("CONTAINER_HOSTNAME")
	if containerHostname == "" {
		return nil, fmt.Errorf("CONTAINER_HOSTNAME is required")
	}
	workingDir := os.Getenv("WORKING_DIR")
	if workingDir == "" {
		return nil, fmt.Errorf("WORKING_DIR is required")
	}
	timezone := os.Getenv("TZ")
	if timezone == "" {
		return nil, fmt.Errorf("TZ is required")
	}
	readonlyRootFS := true

	waitTimeStr := os.Getenv("MANAGER_WAIT_TIME")
	if waitTimeStr == "" {
		return nil, fmt.Errorf("MANAGER_WAIT_TIME is required")
	}
	waitTime, err := time.ParseDuration(waitTimeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid MANAGER_WAIT_TIME: %w", err)
	}

	hostHomesDir := os.Getenv("HOST_HOMES_DIR")
	if hostHomesDir == "" {
		return nil, fmt.Errorf("HOST_HOMES_DIR is required")
	}
	hostShareDir := os.Getenv("HOST_SHARE_DIR")
	if hostShareDir == "" {
		return nil, fmt.Errorf("HOST_SHARE_DIR is required")
	}
	hostReadonlyDir := os.Getenv("HOST_READONLY_DIR")
	if hostReadonlyDir == "" {
		return nil, fmt.Errorf("HOST_READONLY_DIR is required")
	}
	containerShareDir := os.Getenv("CONTAINER_SHARE_DIR")
	if containerShareDir == "" {
		return nil, fmt.Errorf("CONTAINER_SHARE_DIR is required")
	}
	containerReadonlyDir := os.Getenv("CONTAINER_READONLY_DIR")
	if containerReadonlyDir == "" {
		return nil, fmt.Errorf("CONTAINER_READONLY_DIR is required")
	}

	return &config.Config{
		ListenAddr:              listenAddr,
		UserImage:               userImage,
		UserContainerNamePrefix: userContainerNamePrefix,
		NetworkPrefix:           networkPrefix,
		BaseIP:                  baseIP,
		AuthContainerName:       authContainerName,
		AdminUserID:             adminUserID,

		RuntimeUser:          runtimeUser,
		ContainerRuntimeUser: containerRuntimeUser,
		ContainerHostname:    containerHostname,
		WorkingDir:           workingDir,
		Timezone:             timezone,
		ReadOnlyRootFS:       readonlyRootFS,
		ManagerWaitTime:      waitTime,

		HostHomesDir:         hostHomesDir,
		HostShareDir:         hostShareDir,
		HostReadonlyDir:      hostReadonlyDir,
		ContainerShareDir:    containerShareDir,
		ContainerReadonlyDir: containerReadonlyDir,
	}, nil
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
