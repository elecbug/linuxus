package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/client"
	"github.com/elecbug/linuxus/src/internal/common/system_api"
	"github.com/elecbug/linuxus/src/internal/common/user"
)

func (a *App) diagnoseRuntime(report func(string, string, string)) {
	for _, entry := range []struct{ name, path string }{
		{"account storage", filepath.Dir(a.Config.AuthService.Mounts.HostAuthListPath)},
		{"home image storage", a.Config.Volumes.Host.Homes},
		{"shared image storage", filepath.Dir(a.Config.Volumes.Host.Share)},
		{"readonly image storage", filepath.Dir(a.Config.Volumes.Host.Readonly)},
	} {
		total, free, err := system_api.DiskSpace(entry.path)
		if err != nil {
			report("FAIL", entry.name, err.Error())
			continue
		}
		status := "OK"
		if free == 0 {
			status = "FAIL"
		}
		report(status, entry.name, fmt.Sprintf("%s: %.2f GiB free of %.2f GiB", entry.path, float64(free)/(1<<30), float64(total)/(1<<30)))
	}
	if err := a.checkFreeSpace(); err != nil {
		report("FAIL", "storage reserve", err.Error()+"; free space or adjust CAPACITY_MIN_FREE_SPACE")
	}
	api := system_api.NewSystemAPI()
	mounts := []string{a.Config.Volumes.Host.Share, a.Config.Volumes.Host.Readonly}
	users, err := user.TryLoadUsers(a.Config.AuthService.Mounts.HostAuthListPath)
	if err == nil {
		for id := range users {
			mounts = append(mounts, filepath.Join(a.Config.Volumes.Host.Homes, id))
		}
	} else if !os.IsNotExist(err) {
		report("FAIL", "account database", err.Error()+"; inspect AUTH_LIST or retry after account maintenance")
	}
	for _, path := range mounts {
		mounted, err := api.IsMountPoint(path)
		switch {
		case err != nil:
			report("FAIL", "disk mount", err.Error())
		case !mounted:
			report("WARN", "disk mount", path+" is not mounted; run up or ensure-disk before access")
		default:
			report("OK", "disk mount", path)
		}
	}
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return
	}
	defer cli.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, name := range []string{a.Config.AuthService.Container.Name, a.Config.ManagerService.Container.Name} {
		info, err := cli.ContainerInspect(ctx, name)
		if err != nil {
			report("FAIL", name, fmt.Sprintf("%v; start services with up (or systemctl start linuxus)", err))
			continue
		}
		if info.State == nil || !info.State.Running {
			report("FAIL", name, "container is not running; inspect Docker logs and restart services")
			continue
		}
		if info.State.Health != nil && info.State.Health.Status != "healthy" {
			report("FAIL", name, "health status "+info.State.Health.Status+"; inspect Docker logs")
		} else {
			report("OK", name, "running")
		}
	}
	ctxDisk, cancelDisk := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelDisk()
	if err := probeDiskService(ctxDisk, a.diskSocket()); err != nil {
		report("FAIL", "disk service", err.Error()+"; run up or restart linuxus.service")
	} else {
		report("OK", "disk service", "healthy")
	}
}
