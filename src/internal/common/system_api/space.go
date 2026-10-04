package system_api

import (
	"os"
	"path/filepath"
	"syscall"
)

// DiskSpace reports the nearest existing filesystem without creating paths.
func DiskSpace(path string) (total, available uint64, err error) {
	for {
		var stat syscall.Statfs_t
		err = syscall.Statfs(path, &stat)
		if err == nil {
			return stat.Blocks * uint64(stat.Bsize), stat.Bavail * uint64(stat.Bsize), nil
		}
		parent := filepath.Dir(path)
		if !os.IsNotExist(err) || parent == path {
			return 0, 0, err
		}
		path = parent
	}
}
