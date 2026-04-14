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
	"github.com/moby/term"
)

func (a *App) buildRuntimeImages() error {
	fmt.Println("[+] Building runtime images...")

	if a.DockerClient == nil {
		return fmt.Errorf("Docker client is not initialized")
	}

	if err := a.buildImage(a.Config.AuthService.SourceDir, a.authImageName(), nil); err != nil {
		return fmt.Errorf("failed to build auth image: %w", err)
	}

	if err := a.buildImage(a.Config.UserService.SourceDir, a.userImageName(), map[string]*string{
		"CONTAINER_RUNTIME_USER": strPtr(a.Config.UserService.Container.Runtime.User),
	}); err != nil {
		return fmt.Errorf("failed to build user image: %w", err)
	}

	return nil
}

func (a *App) buildImage(sourceDir string, tag string, buildArgs map[string]*string) error {
	buildCtx, err := tarBuildContext(sourceDir)
	if err != nil {
		return err
	}

	resp, err := a.DockerClient.ImageBuild(a.Context, buildCtx, build.ImageBuildOptions{
		Tags:       []string{tag},
		Dockerfile: "Dockerfile",
		Remove:     true,
		BuildArgs:  buildArgs,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fd, isTerm := term.GetFdInfo(os.Stdout)

	err = jsonmessage.DisplayJSONMessagesStream(
		resp.Body,
		os.Stdout,
		fd,
		isTerm,
		nil,
	)
	if err != nil {
		return err
	}

	return nil
}

func tarBuildContext(dir string) (io.Reader, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

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

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
		}

		return nil
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
