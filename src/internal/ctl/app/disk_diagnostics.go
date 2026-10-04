package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/elecbug/linuxus/src/internal/common/convert"
	"github.com/elecbug/linuxus/src/internal/common/system_api"
	"github.com/elecbug/linuxus/src/internal/common/user"
)

func (a *App) checkFreeSpace() error {
	if a.Config.Capacity.MinFreeSpace == "" {
		return nil
	}
	minimum, err := convert.BytesFromString(a.Config.Capacity.MinFreeSpace)
	if err != nil {
		return err
	}
	for _, path := range []string{a.Config.Volumes.Host.Homes, filepath.Dir(a.Config.Volumes.Host.Share), filepath.Dir(a.Config.Volumes.Host.Readonly)} {
		_, available, err := system_api.DiskSpace(path)
		if err != nil {
			return err
		}
		if available < uint64(minimum) {
			return fmt.Errorf("insufficient host free space at %s: %d bytes available, %d required", path, available, minimum)
		}
	}
	return nil
}

func (a *App) registerDiskDiagnostics(mux *http.ServeMux) {
	mux.HandleFunc("GET /capacity", func(w http.ResponseWriter, r *http.Request) {
		if err := a.checkFreeSpace(); err != nil {
			http.Error(w, err.Error(), 503)
			return
		}
		w.WriteHeader(200)
	})
	mux.HandleFunc("GET /usage", func(w http.ResponseWriter, r *http.Request) {
		users, err := user.LoadUsers(a.Config.AuthService.Mounts.HostAuthListPath)
		if err != nil {
			http.Error(w, "cannot load accounts", 500)
			return
		}
		rows := make(map[string]any)
		for id := range users {
			path := filepath.Join(a.Config.Volumes.Host.Homes, id)
			mounted, err := a.systemAPI.IsMountPoint(path)
			if err != nil {
				rows[id] = map[string]any{"error": err.Error()}
				continue
			}
			row := map[string]any{"mounted": mounted}
			if mounted {
				total, free, err := system_api.DiskSpace(path)
				if err == nil {
					row["total_bytes"] = total
					row["available_bytes"] = free
					row["used_bytes"] = total - free
				}
			}
			rows[id] = row
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rows)
	})
}
