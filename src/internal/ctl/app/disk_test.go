package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/elecbug/linuxus/src/internal/common/system_api"
)

type cleanupAPI struct {
	system_api.API
	events                []string
	unmountErr, detachErr error
	nested                []string
}

func (f *cleanupAPI) MountPointsUnder(string) ([]string, error) {
	return append([]string(nil), f.nested...), nil
}

func (f *cleanupAPI) IsMountPoint(p string) (bool, error) { return true, nil }
func (f *cleanupAPI) Unmount(p string) error {
	f.events = append(f.events, "unmount:"+p)
	return f.unmountErr
}
func (f *cleanupAPI) FindLoopDevicesForImages(p string) ([]string, error) {
	f.events = append(f.events, "find:"+p)
	return []string{"/dev/loop42"}, nil
}
func (f *cleanupAPI) DetachLoopDevice(p string) error {
	f.events = append(f.events, "detach:"+p)
	return f.detachErr
}
func (f *cleanupAPI) RemoveAll(p string) error {
	f.events = append(f.events, "remove-dir:"+p)
	return nil
}
func (f *cleanupAPI) Remove(p string) error {
	f.events = append(f.events, "remove-file:"+p)
	return nil
}

func TestUserDiskCleanupStopsBeforeDeletingAttachedData(t *testing.T) {
	for _, step := range []string{"unmount", "detach", "success"} {
		t.Run(step, func(t *testing.T) {
			api := &cleanupAPI{}
			failure := errors.New("busy")
			if step == "unmount" {
				api.unmountErr = failure
			}
			if step == "detach" {
				api.detachErr = failure
			}
			a := &App{systemAPI: api}
			a.Config.AuthService.Mounts.HostAuthListPath = filepath.Join(t.TempDir(), "AUTH_LIST")
			a.Config.Volumes.Host.Homes = "/deployment/volumes/homes"
			err := a.cleanVolumeUser("alice")
			want := []string{"unmount:/deployment/volumes/homes/alice"}
			if step != "unmount" {
				want = append(want, "find:/deployment/volumes/homes/alice.img", "detach:/dev/loop42")
			}
			if step == "success" {
				want = append(want, "remove-dir:/deployment/volumes/homes/alice", "remove-file:/deployment/volumes/homes/alice.img")
				if err != nil {
					t.Fatal(err)
				}
			} else if !errors.Is(err, failure) {
				t.Fatalf("lost cleanup failure: %v", err)
			}
			if !reflect.DeepEqual(api.events, want) {
				t.Fatalf("operations=%v, want %v", api.events, want)
			}
		})
	}
}

func TestDiskOperationsRejectInvalidUserPaths(t *testing.T) {
	a := &App{}
	for _, id := range []string{"", "..", "../outside", "/absolute", "a/b"} {
		if err := a.cleanVolumeUser(id); err == nil {
			t.Fatalf("accepted cleanup ID %q", id)
		}
		if err := a.createUserDisk(id, false); err == nil {
			t.Fatalf("accepted disk ID %q", id)
		}
	}
}

type failingDiskAPI struct {
	recordingDiskAPI
	formatErr, chownErr, unmountErr, detachErr error
	nested                                     []string
	unmounted                                  int
}

func (f *failingDiskAPI) FormatExt4(path string) error {
	if f.formatErr != nil {
		return f.formatErr
	}
	return f.recordingDiskAPI.FormatExt4(path)
}
func (f *failingDiskAPI) Remove(path string) error     { delete(f.images, path); return nil }
func (f *failingDiskAPI) Chown(string, int, int) error { return f.chownErr }
func (f *failingDiskAPI) Unmount(path string) error {
	f.unmounted++
	if f.unmountErr == nil {
		delete(f.mounted, path)
	}
	return f.unmountErr
}
func (f *failingDiskAPI) DetachLoopDevice(path string) error { f.detached++; return f.detachErr }

func diskFailureFixture(t *testing.T, shared bool) (*failingDiskAPI, string, func() error) {
	t.Helper()
	api := &failingDiskAPI{recordingDiskAPI: recordingDiskAPI{
		images: map[string]int64{}, formatted: map[string]int{}, mounted: map[string]bool{},
	}}
	a := &App{systemAPI: api}
	a.Config.Volumes.DiskLimit = "8M"
	a.Config.UserService.Limits.User.Disk = "8M"
	a.Config.Volumes.Host.Homes = filepath.Join(t.TempDir(), "homes")
	if shared {
		path := filepath.Join(t.TempDir(), "share")
		return api, path + ".img", func() error { return a.createSharedDisk(path) }
	}
	return api, filepath.Join(a.Config.Volumes.Host.Homes, "alice.img"), func() error { return a.createUserDisk("alice", false) }
}

func TestFailedFormatCanBeRetried(t *testing.T) {
	for _, shared := range []bool{false, true} {
		t.Run(map[bool]string{true: "shared", false: "user"}[shared], func(t *testing.T) {
			api, image, create := diskFailureFixture(t, shared)
			failure := errors.New("mkfs failed")
			api.formatErr = failure
			if err := create(); !errors.Is(err, failure) {
				t.Fatal(err)
			}
			if _, exists := api.images[image]; exists {
				t.Fatal("incomplete image prevents a successful retry")
			}
			api.formatErr = nil
			if err := create(); err != nil {
				t.Fatal(err)
			}
			if api.formatted[image] != 1 {
				t.Fatal("retry did not format a new image")
			}
		})
	}
}

func TestMountRollbackPreservesCleanupErrors(t *testing.T) {
	for _, shared := range []bool{false, true} {
		for _, busy := range []bool{false, true} {
			t.Run(fmt.Sprintf("shared=%t/unmount_busy=%t", shared, busy), func(t *testing.T) {
				api, _, create := diskFailureFixture(t, shared)
				original, cleanup := errors.New("chown failed"), errors.New("cleanup failed")
				api.chownErr = original
				if busy {
					api.unmountErr = cleanup
				} else {
					api.detachErr = cleanup
				}
				err := create()
				if !errors.Is(err, original) || !errors.Is(err, cleanup) {
					t.Fatalf("lost failure details: %v", err)
				}
				if busy && api.detached != 0 {
					t.Fatal("detached loop device while filesystem was mounted")
				}
				if !busy && api.detached != 1 {
					t.Fatal("did not attempt to release loop device")
				}
			})
		}
	}
}

func TestUserDiskCleanupUnmountsNestedDisksFirst(t *testing.T) {
	root := "/deployment/volumes/homes/alice"
	api := &cleanupAPI{nested: []string{root + "/bind", root + "/bind/deeper"}}
	a := &App{systemAPI: api}
	a.Config.Volumes.Host.Homes = filepath.Dir(root)
	if err := a.cleanVolumeUserUnlocked("alice"); err != nil {
		t.Fatal(err)
	}
	want := []string{"unmount:" + root + "/bind/deeper", "unmount:" + root + "/bind", "unmount:" + root}
	if len(api.events) < len(want) || !reflect.DeepEqual(api.events[:len(want)], want) {
		t.Fatalf("unsafe unmount order: %v", api.events)
	}
}
