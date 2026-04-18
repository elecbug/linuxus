package app

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

func (a *App) existDockerContainer(name string) (bool, error) {
	cli := a.dockerClient
	if cli == nil {
		return false, fmt.Errorf("Docker client is not initialized")
	}

	summary, err := cli.ContainerList(a.context, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.KeyValuePair{Key: "name", Value: "^/" + name + "$"}),
	})
	if err != nil {
		return false, err
	}
	return len(summary) > 0, nil
}
