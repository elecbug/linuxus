package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/elecbug/linuxus/src/internal/common/config"
	"github.com/elecbug/linuxus/src/internal/common/user"
	"github.com/elecbug/linuxus/src/internal/ctl/cli"
	"github.com/elecbug/linuxus/src/internal/ctl/log"
)

func (a *App) ListUsers(output io.Writer) error {
	ids := make([]string, 0, len(a.UserIDs))
	for id := range a.UserIDs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	fmt.Fprintln(output, "USER\tLOCKED\tCLASS\tTEMPLATE")
	for _, id := range ids {
		record, err := user.ParseAccount(a.UserIDs[id])
		if err != nil {
			return err
		}
		fmt.Fprintf(output, "%s\t%t\t%s\t%s\n", id, record.Locked || record.Maintenance, record.Class, record.Template)
	}
	return nil
}

func (a *App) ListTemplates(output io.Writer) error {
	templates, classes, err := config.ParseTemplates(&a.Config)
	if err != nil {
		return err
	}
	fmt.Fprintf(output, "default\t%s\n", a.userImageName())
	names := make([]string, 0, len(templates))
	for name := range templates {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		entry := templates[name]
		if entry.Image == "default" {
			entry.Image = a.userImageName()
		}
		fmt.Fprintf(output, "%s\t%s\t%s\n", name, entry.Image, entry.Seed)
	}
	names = nil
	for name := range classes {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(output, "class %s -> %s\n", name, classes[name])
	}
	return nil
}

func (a *App) ServiceAccount(command string, params *cli.Parameters) error {
	id := params.Params["user"]
	if command == "recover-user" {
		return a.RecoverUser(id)
	}
	actions := map[string]string{"lock-user": "lock", "unlock-user": "unlock", "reset-password": "password", "disconnect-user": "disconnect", "assign-template": "template", "assign-class": "class"}
	action := actions[command]
	value := ""
	if action == "password" {
		password, err := log.InputPassword("New password for %s: ", id)
		if err != nil {
			return err
		}
		confirmation, err := log.InputPassword("Confirm password: ")
		if err != nil {
			return err
		}
		if password != confirmation {
			return fmt.Errorf("passwords do not match")
		}
		value = password
	}
	if action == "template" || action == "class" {
		templates, classes, err := config.ParseTemplates(&a.Config)
		if err != nil {
			return err
		}
		value = params.Params[action]
		if action == "template" {
			if _, ok := templates[value]; !ok && value != "default" {
				return fmt.Errorf("unknown template %s", value)
			}
		} else if _, ok := classes[value]; !ok {
			return fmt.Errorf("unknown class %s", value)
		}
	}
	if err := user.UpdateAccount(a.Config.AuthService.Mounts.HostAuthListPath, id, action, value); err != nil {
		return err
	}
	if action != "unlock" {
		if err := a.stopUserRuntime(id); err != nil {
			return fmt.Errorf("account updated; could not terminate runtime: %w", err)
		}
	}
	log.Log(log.DETAIL_PREFIX, "%s completed for %s; home data preserved.", command, id)
	return nil
}

// Stop through Manager's preparation lock so in-flight startup cannot recreate
// a container after a maintenance operation has stopped it.
func (a *App) stopUserRuntime(id string) error {
	ctx, cancel := context.WithTimeout(a.context, 50*time.Second)
	defer cancel()
	manager, err := a.dockerClient.ContainerInspect(ctx, a.Config.ManagerService.Container.Name)
	if err == nil && manager.State != nil && manager.State.Running {
		if manager.NetworkSettings == nil {
			return fmt.Errorf("Manager has no network settings")
		}
		network := manager.NetworkSettings.Networks[a.Config.ManagerService.Container.Network]
		if network == nil || network.IPAddress == "" {
			return fmt.Errorf("Manager IP is unavailable")
		}
		data, _ := json.Marshal(map[string]string{"user_id": id, "action": "stop"})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+network.IPAddress+":5959/admin/user", bytes.NewReader(data))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Manager-Session-Secret", a.Config.ManagerService.Security.SessionSecret)
		client := &http.Client{Timeout: 50 * time.Second, Transport: &http.Transport{Proxy: nil}}
		defer client.CloseIdleConnections()
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			return fmt.Errorf("Manager stop returned %d: %s", resp.StatusCode, body)
		}
		return nil
	}
	if err != nil && !errdefs.IsNotFound(err) {
		return err
	}
	name := a.Config.UserService.Container.NamePrefix + id
	timeout := 10
	if err := a.dockerClient.ContainerStop(ctx, name, container.StopOptions{Timeout: &timeout}); err != nil && !errdefs.IsNotFound(err) {
		return err
	}
	if err := a.dockerClient.ContainerRemove(ctx, name, container.RemoveOptions{Force: true}); err != nil && !errdefs.IsNotFound(err) {
		return err
	}
	return nil
}
