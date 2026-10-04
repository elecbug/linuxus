package app

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"runtime"
	"strconv"

	"github.com/docker/docker/api/types/build"
	assets "github.com/elecbug/linuxus/src/docker"
	"github.com/elecbug/linuxus/src/internal/ctl/log"
)

// buildRuntimeImages assembles runtime images from the executable and embedded assets.
func (a *App) buildRuntimeImages() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("runtime images require a Linux linuxusctl binary")
	}
	if a.dockerClient == nil {
		return fmt.Errorf("Docker client is not initialized")
	}
	log.Log(log.DETAIL_PREFIX, "Building runtime images...")
	if err := a.buildImage("auth.Dockerfile", a.authImageName(), nil); err != nil {
		return fmt.Errorf("failed to build auth image: %w", err)
	}
	if err := a.buildImage("manager.Dockerfile", a.managerImageName(), nil); err != nil {
		return fmt.Errorf("failed to build manager image: %w", err)
	}
	uid := strconv.Itoa(a.Config.UserService.Runtime.UID)
	gid := strconv.Itoa(a.Config.UserService.Runtime.GID)
	if err := a.buildImage("user.Dockerfile", a.userImageName(), map[string]*string{
		"CONTAINER_RUNTIME_USER": &a.Config.UserService.Runtime.LinuxUsername,
		"CONTAINER_UID":          &uid,
		"CONTAINER_GID":          &gid,
	}); err != nil {
		return fmt.Errorf("failed to build user image: %w", err)
	}
	return nil
}

func (a *App) buildImage(dockerfile, tag string, buildArgs map[string]*string) error {
	buildCtx, err := tarRuntimeContext(dockerfile, a.execPath)
	if err != nil {
		return fmt.Errorf("failed to create Docker build context: %w", err)
	}
	resp, err := a.dockerClient.ImageBuild(a.context, buildCtx, build.ImageBuildOptions{
		Tags:       []string{tag},
		Dockerfile: "Dockerfile",
		Platform:   "linux/" + runtime.GOARCH,
		Remove:     true,
		BuildArgs:  buildArgs,
	})
	if err != nil {
		return fmt.Errorf("failed to build image %s: %w", tag, err)
	}
	defer resp.Body.Close()
	return log.DockerBuildLog(log.DETAIL_PREFIX, resp.Body, tag)
}

// tarRuntimeContext uses an explicit allowlist so credentials, source and disks
// cannot enter a Docker build context. The running binary also serves auth/manager.
func tarRuntimeContext(dockerfile, executable string) (io.Reader, error) {
	switch dockerfile {
	case "auth.Dockerfile", "manager.Dockerfile", "user.Dockerfile":
	default:
		return nil, fmt.Errorf("unknown runtime Dockerfile: %s", dockerfile)
	}
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	add := func(name string, mode int64, data []byte) error {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: int64(len(data))}); err != nil {
			return err
		}
		_, err := tw.Write(data)
		return err
	}
	definition, err := assets.Files.ReadFile(dockerfile)
	if err != nil {
		return nil, err
	}
	if err := add("Dockerfile", 0644, definition); err != nil {
		return nil, err
	}
	if dockerfile == "user.Dockerfile" {
		script, err := assets.Files.ReadFile("start.sh")
		if err != nil {
			return nil, err
		}
		if err := add("start.sh", 0755, script); err != nil {
			return nil, err
		}
		if err := fs.WalkDir(assets.Files, "templates", func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			data, err := assets.Files.ReadFile(path)
			if err != nil {
				return err
			}
			return add(path, 0644, data)
		}); err != nil {
			return nil, err
		}
	} else {
		binary, err := os.Open(executable)
		if err != nil {
			return nil, err
		}
		defer binary.Close()
		info, err := binary.Stat()
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("executable is not a regular file: %s", executable)
		}
		if err := tw.WriteHeader(&tar.Header{Name: "linuxusctl", Mode: 0755, Size: info.Size()}); err != nil {
			return nil, err
		}
		if _, err := io.Copy(tw, binary); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf, nil
}

// authImageName returns the auth runtime image tag.
func (a *App) authImageName() string {
	return a.Config.AuthService.Container.Name + ":runtime"
}

// userImageName returns the user base runtime image tag.
func (a *App) userImageName() string {
	return a.Config.UserService.Container.NamePrefix + "base:runtime"
}

// managerImageName returns the manager runtime image tag.
func (a *App) managerImageName() string {
	return a.Config.ManagerService.Container.Name + ":runtime"
}
