package app

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/elecbug/linuxus/ctl/internal/format"
)

// buildRuntimeImages builds all runtime images required by services.
func (a *App) buildRuntimeImages() error {
	format.Log(format.DETAIL_PREFIX, "Building runtime images...")

	if a.dockerClient == nil {
		return fmt.Errorf("Docker client is not initialized")
	}

	if err := a.buildImage(a.Config.AuthService.SourceDir, a.authImageName(), nil); err != nil {
		return fmt.Errorf("failed to build auth image: %w", err)
	}

	if err := a.buildImage(a.Config.ManagerService.SourceDir, a.managerImageName(), nil); err != nil {
		return fmt.Errorf("failed to build manager image: %w", err)
	}

	if err := a.buildImage(a.Config.UserService.SourceDir, a.userImageName(), map[string]*string{
		"CONTAINER_RUNTIME_USER": &a.Config.UserService.Runtime.LinuxUsername,
	}); err != nil {
		return fmt.Errorf("failed to build user image: %w", err)
	}

	return nil
}

// buildImage builds a Docker image from a source directory.
func (a *App) buildImage(sourceDir string, tag string, buildArgs map[string]*string) error {
	buildCtx, err := tarBuildContext(sourceDir)
	if err != nil {
		return err
	}

	resp, err := a.dockerClient.ImageBuild(a.context, buildCtx, build.ImageBuildOptions{
		Tags:       []string{tag},
		Dockerfile: "Dockerfile",
		Remove:     true,
		BuildArgs:  buildArgs,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var logBuf bytes.Buffer

	err = jsonmessage.DisplayJSONMessagesStream(
		resp.Body,
		&logBuf,
		0,
		false,
		nil,
	)

	if inErr := format.DockerBuildLog(format.DETAIL_PREFIX, logBuf, tag); inErr != nil {
		return fmt.Errorf("failed to process Docker build log for image %s: %w", tag, inErr)
	}

	if err != nil {
		return fmt.Errorf("failed to build image %s: %w", tag, err)
	}

	return nil
}

// tarBuildContext creates an in-memory tar archive for Docker build context.
func tarBuildContext(dir string) (io.Reader, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	visited := make(map[string]bool)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		if info.Mode()&os.ModeSymlink != 0 {
			target, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}

			if visited[target] {
				return nil
			}
			visited[target] = true

			targetInfo, err := os.Stat(target)
			if err != nil {
				return err
			}

			if targetInfo.IsDir() {
				return filepath.Walk(target, func(p string, ti os.FileInfo, err error) error {
					if err != nil {
						return err
					}

					relSub, err := filepath.Rel(target, p)
					if err != nil {
						return err
					}

					newPath := filepath.Join(relPath, relSub)
					return addFileToTar(tw, p, newPath, ti)
				})
			}

			return addFileToTar(tw, target, relPath, targetInfo)
		}

		return addFileToTar(tw, path, relPath, info)
	})

	if err != nil {
		_ = tw.Close()
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return buf, nil
}

func addFileToTar(tw *tar.Writer, realPath, tarPath string, info os.FileInfo) error {
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = filepath.ToSlash(tarPath)

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	if info.Mode().IsRegular() {
		f, err := os.Open(realPath)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(tw, f); err != nil {
			return err
		}
	}

	return nil
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
