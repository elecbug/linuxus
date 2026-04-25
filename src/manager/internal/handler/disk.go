package handler

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/elecbug/linuxus/pkg/system_api"
)

// systemAPI provides an abstraction layer for OS-specific operations related to disk management.
var systemAPI system_api.API = system_api.NewSystemAPI()

// PrepareUserDisks creates and mounts shared/admin/user disk images.
func (s *Server) PrepareUserDisks(userID string) error {
	if err := systemAPI.MkdirAll(s.cfg.HostHomesDir, 0755); err != nil {
		return err
	}

	if err := s.createSharedDisk(s.cfg.HostShareDir); err != nil {
		return err
	}
	if err := s.createSharedDisk(s.cfg.HostReadonlyDir); err != nil {
		return err
	}

	if err := s.createUserDisk(userID, s.cfg.AdminUserID == userID); err != nil {
		return err
	}

	return nil
}

// createSharedDisk creates and mounts a shared loopback disk at the target path.
func (s *Server) createSharedDisk(path string) error {
	sizeStr := s.cfg.ShareDiskLimit
	size, err := stringToBytes(sizeStr)
	if err != nil {
		return fmt.Errorf("invalid disk size for shared disk: %w", err)
	}
	if size <= 1024*1024 {
		return fmt.Errorf("volumes.disk_limit must be at least 1MB, got %d", size)
	}

	parentDir := filepath.Dir(path)
	name := filepath.Base(path)

	if err := systemAPI.MkdirAll(parentDir, 0755); err != nil {
		return err
	}

	img := filepath.Join(parentDir, name+".img")
	mountPoint := path

	if mounted, err := systemAPI.IsMountPoint(mountPoint); err != nil {
		return err
	} else if mounted {
		return nil
	}

	if exists, err := systemAPI.Exists(img); err != nil {
		return fmt.Errorf("failed to stat image file %s: %w", img, err)
	} else if !exists {
		if err := systemAPI.CreateEmptyFile(img, size); err != nil {
			return err
		}
		if err := systemAPI.FormatExt4(img); err != nil {
			return err
		}
	}

	if err := systemAPI.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point %s: %w", mountPoint, err)
	}

	loopdev, err := systemAPI.AttachLoopDevice(img)
	if err != nil {
		return err
	}

	mounted := false
	defer func() {
		if err == nil {
			return
		}
		if mounted {
			_ = systemAPI.Unmount(mountPoint)
		}
		_ = systemAPI.DetachLoopDevice(loopdev)
	}()

	if err = systemAPI.Mount(loopdev, mountPoint); err != nil {
		return err
	}
	mounted = true

	splits := strings.Split(s.cfg.RuntimeUser, ":")
	uid, gid := 0, 0
	if len(splits) == 2 {
		uid, err = strconv.Atoi(splits[0])
		if err != nil {
			return fmt.Errorf("invalid UID in runtime_user: %w", err)
		}
		gid, err = strconv.Atoi(splits[1])
		if err != nil {
			return fmt.Errorf("invalid GID in runtime_user: %w", err)
		}
	}

	if err = systemAPI.Chown(
		mountPoint,
		uid,
		gid,
	); err != nil {
		return err
	}

	if err = systemAPI.Chmod(mountPoint, 0755); err != nil {
		return err
	}

	return nil
}

// createUserDisk creates and mounts a per-user loopback disk.
func (s *Server) createUserDisk(userID string, isAdmin bool) error {
	sizeStr := s.cfg.UserDiskLimit
	if isAdmin {
		sizeStr = s.cfg.AdminDiskLimit
	}

	size, err := stringToBytes(sizeStr)
	if err != nil {
		return fmt.Errorf("invalid disk size for %s: %w", userID, err)
	}
	if size <= 1024*1024 {
		userMode := "user"
		if isAdmin {
			userMode = "admin"
		}
		return fmt.Errorf("disk limit for %s must be at least 1MB, got %d", userMode, size)
	}

	img := filepath.Join(s.cfg.HostHomesDir, userID+".img")
	mountPoint := filepath.Join(s.cfg.HostHomesDir, userID)

	if mounted, err := systemAPI.IsMountPoint(mountPoint); err != nil {
		return err
	} else if mounted {
		return nil
	}

	if exists, err := systemAPI.Exists(img); err != nil {
		return fmt.Errorf("failed to stat image file %s: %w", img, err)
	} else if !exists {
		if err := systemAPI.CreateEmptyFile(img, size); err != nil {
			return err
		}
		if err := systemAPI.FormatExt4(img); err != nil {
			return err
		}
	}

	if err := systemAPI.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point %s: %w", mountPoint, err)
	}

	loopdev, err := systemAPI.AttachLoopDevice(img)
	if err != nil {
		return err
	}

	mounted := false
	defer func() {
		if err == nil {
			return
		}
		if mounted {
			_ = systemAPI.Unmount(mountPoint)
		}
		_ = systemAPI.DetachLoopDevice(loopdev)
	}()

	if err = systemAPI.Mount(loopdev, mountPoint); err != nil {
		return err
	}
	mounted = true

	splits := strings.Split(s.cfg.RuntimeUser, ":")
	uid, gid := 0, 0
	if len(splits) == 2 {
		uid, err = strconv.Atoi(splits[0])
		if err != nil {
			return fmt.Errorf("invalid UID in runtime_user: %w", err)
		}
		gid, err = strconv.Atoi(splits[1])
		if err != nil {
			return fmt.Errorf("invalid GID in runtime_user: %w", err)
		}
	}

	if err = systemAPI.Chown(
		mountPoint,
		uid,
		gid,
	); err != nil {
		return err
	}

	if err = systemAPI.Chmod(mountPoint, 0755); err != nil {
		return err
	}

	return nil
}

// listMountedDirsDeepestFirst returns mounted directories under root from deepest to shallowest.
func (s *Server) listMountedDirsDeepestFirst(root string) ([]string, error) {
	var dirs []string

	exists, err := systemAPI.Exists(root)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if path == root {
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		mounted, err := systemAPI.IsMountPoint(path)
		if err != nil {
			return nil
		}
		if mounted {
			dirs = append(dirs, path)
			return fs.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(dirs, func(i, j int) bool {
		return len(dirs[i]) > len(dirs[j])
	})

	return dirs, nil
}

// stringToBytes converts a memory string to bytes.
func stringToBytes(v string) (int64, error) {
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
