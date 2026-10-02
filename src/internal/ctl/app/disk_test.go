package app

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/elecbug/linuxus/src/internal/common/system_api"
)

type cleanupAPI struct {
	system_api.API
	events                []string
	unmountErr, detachErr error
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
