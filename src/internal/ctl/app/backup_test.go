package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type restoreAPI struct {
	recordingDiskAPI
	failOnce bool
}

func (f *restoreAPI) Mount(source, target string) error {
	if f.failOnce {
		f.failOnce = false
		return errors.New("injected mount failure")
	}
	return f.recordingDiskAPI.Mount(source, target)
}
func (f *restoreAPI) MountPointsUnder(string) ([]string, error)         { return nil, nil }
func (f *restoreAPI) FindLoopDevicesForImages(string) ([]string, error) { return nil, nil }

func TestRestorePreservesOriginalOnMountFailure(t *testing.T) {
	for _, mode := range []string{"success", "mount-failure", "no-replace"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			image := filepath.Join(root, "alice.img")
			prepared := filepath.Join(root, ".restore-new")
			os.WriteFile(image, []byte("original student data"), 0600)
			os.WriteFile(prepared, []byte("restored student data"), 0600)
			api := &restoreAPI{recordingDiskAPI: recordingDiskAPI{images: map[string]int64{image: 8 << 20}, formatted: map[string]int{}, mounted: map[string]bool{}}, failOnce: mode == "mount-failure"}
			a := &App{systemAPI: api}
			a.Config.Volumes.Host.Homes = root
			a.Config.UserService.Limits.User.Disk = "8M"
			err := a.restorePreparedImage("alice", prepared, mode != "no-replace")
			if (err == nil) != (mode == "success") {
				t.Fatalf("mode=%s error=%v", mode, err)
			}
			data, _ := os.ReadFile(image)
			want := "original student data"
			if mode == "success" {
				want = "restored student data"
			}
			if string(data) != want {
				t.Fatalf("data lost: %q", data)
			}
			if len(api.formatted) != 0 {
				t.Fatal("restore formatted an existing image")
			}
			if _, err := os.Stat(a.previousUserImage("alice")); !os.IsNotExist(err) {
				t.Fatal("successful recovery left an unexpected previous image")
			}
		})
	}
}

func recoveryFixture(t *testing.T) (*App, *restoreAPI, string) {
	t.Helper()
	root := t.TempDir()
	image := filepath.Join(root, "alice.img")
	api := &restoreAPI{recordingDiskAPI: recordingDiskAPI{images: map[string]int64{image: 8 << 20}, formatted: map[string]int{}, mounted: map[string]bool{}}}
	a := &App{systemAPI: api}
	a.Config.Volumes.Host.Homes = root
	a.Config.UserService.Limits.User.Disk = "8M"
	return a, api, image
}

func writeRecoveryFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestRecoverInterruptedImageReplacement(t *testing.T) {
	for _, stage := range []string{"before-replacement", "after-replacement", "missing-current", "after-commit"} {
		t.Run(stage, func(t *testing.T) {
			a, api, image := recoveryFixture(t)
			writeRecoveryFile(t, image, "original data")
			if stage != "after-commit" {
				if err := os.Link(image, a.previousUserImage("alice")); err != nil {
					t.Fatal(err)
				}
			}
			if stage == "missing-current" {
				if err := os.Remove(image); err != nil {
					t.Fatal(err)
				}
			} else if stage == "after-replacement" || stage == "after-commit" {
				prepared := filepath.Join(a.Config.Volumes.Host.Homes, ".restore-new")
				writeRecoveryFile(t, prepared, "replacement data")
				if err := os.Rename(prepared, image); err != nil {
					t.Fatal(err)
				}
			}
			if err := a.recoverUserImage("alice"); err != nil {
				t.Fatal(err)
			}
			want := "original data"
			if stage == "after-commit" {
				want = "replacement data"
			}
			data, err := os.ReadFile(image)
			if err != nil || string(data) != want {
				t.Fatalf("recovery lost data: %q, %v", data, err)
			}
			if _, err := os.Lstat(a.previousUserImage("alice")); !os.IsNotExist(err) {
				t.Fatal("recovery left an obsolete original link")
			}
			if len(api.formatted) != 0 || !api.mounted[filepath.Join(a.Config.Volumes.Host.Homes, "alice")] {
				t.Fatal("recovery formatted data or failed to mount the recovered image")
			}
		})
	}
}

func TestRecoveryNeverCreatesMissingDataOrFollowsSymlinks(t *testing.T) {
	for _, mode := range []string{"missing", "empty", "symlink", "invalid-previous"} {
		t.Run(mode, func(t *testing.T) {
			a, api, image := recoveryFixture(t)
			delete(api.images, image)
			switch mode {
			case "empty":
				writeRecoveryFile(t, image, "")
			case "symlink":
				other := filepath.Join(a.Config.Volumes.Host.Homes, "other.img")
				writeRecoveryFile(t, other, "unrelated data")
				if err := os.Symlink(other, image); err != nil {
					t.Fatal(err)
				}
			case "invalid-previous":
				writeRecoveryFile(t, image, "current data")
				if err := os.Symlink(image, a.previousUserImage("alice")); err != nil {
					t.Fatal(err)
				}
			}
			if err := a.recoverUserImage("alice"); err == nil {
				t.Fatal("invalid recovery source was accepted")
			}
			if len(api.images) != 0 || len(api.formatted) != 0 || len(api.mounted) != 0 {
				t.Fatal("invalid recovery changed disks")
			}
		})
	}
}

func TestRestoreRefusesToOverwritePendingRecovery(t *testing.T) {
	a, api, image := recoveryFixture(t)
	writeRecoveryFile(t, image, "current data")
	writeRecoveryFile(t, a.previousUserImage("alice"), "retained original data")
	prepared := filepath.Join(a.Config.Volumes.Host.Homes, ".restore-new")
	writeRecoveryFile(t, prepared, "another replacement")
	if err := a.restorePreparedImage("alice", prepared, true); err == nil {
		t.Fatal("new restore overwrote pending recovery")
	}
	data, err := os.ReadFile(a.previousUserImage("alice"))
	if err != nil || string(data) != "retained original data" || len(api.mounted) != 0 {
		t.Fatal("pending recovery data changed")
	}
}
