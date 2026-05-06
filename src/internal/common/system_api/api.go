package system_api

import (
	"os"
	"runtime"
)

// API defines an interface for OS-specific operations needed by the application.
type API interface {
	MkdirAll(path string, mode os.FileMode) error
	Chown(path string, uid int, gid int) error
	Chmod(path string, mode os.FileMode) error

	Exists(path string) (bool, error)
	CreateEmptyFile(path string, sizeBytes int64) error
	RemoveAll(path string) error
	Remove(path string) error
	FormatExt4(path string) error

	IsMountPoint(path string) (bool, error)
	AttachLoopDevice(imagePath string) (string, error)
	DetachLoopDevice(loopDevice string) error
	Mount(source string, target string) error
	Unmount(target string) error
	FindLoopDevicesForImages(dir string) ([]string, error)
}

// NewSystemAPI returns a SystemAPI implementation based on the current OS.
func NewSystemAPI() API {
	switch runtime.GOOS {
	case "linux":
		return LinuxSystemAPI{}
	default:
		return UnsupportedSystemAPI{OS: runtime.GOOS}
	}
}
