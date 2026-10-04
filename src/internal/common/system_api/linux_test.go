package system_api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoopDevicesMatchExactImageOrDirectory(t *testing.T) {
	data := []byte(`{"loopdevices":[
        {"name":"/dev/loop1","back-file":"/volumes/homes/alice.img"},
        {"name":"/dev/loop2","back-file":"/volumes/homes/bob.img"},
        {"name":"/dev/loop3","back-file":"/volumes/homes-other/alice.img"},
        {"name":"/dev/loop4","back-file":"/volumes/disk (copy).img"}
    ]}`)
	for _, tc := range []struct {
		path string
		want []string
	}{
		{"/volumes/homes/alice.img", []string{"/dev/loop1"}},
		{"/volumes/homes", []string{"/dev/loop1", "/dev/loop2"}},
		{"/volumes/disk (copy).img", []string{"/dev/loop4"}},
		{"/volumes/homes/alice", nil},
	} {
		got, err := loopDevicesForPath(data, tc.path)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%s: %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestLoopDevicesResolveVolumeSymlink(t *testing.T) {
	root := t.TempDir()
	storage := t.TempDir()
	image := filepath.Join(storage, "alice.img")
	if err := os.WriteFile(image, nil, 0600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "volumes")
	if err := os.Symlink(storage, alias); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(map[string]any{"loopdevices": []map[string]string{{"name": "/dev/loop9", "back-file": image}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{alias, filepath.Join(alias, "alice.img")} {
		got, err := loopDevicesForPath(data, path)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []string{"/dev/loop9"}) {
			t.Fatalf("symlink %s: %v", path, got)
		}
	}
}

func TestCreateEmptyFilePreservesExistingData(t *testing.T) {
	api := LinuxSystemAPI{}
	path := filepath.Join(t.TempDir(), "disk.img")
	if err := os.WriteFile(path, []byte("existing disk data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := api.CreateEmptyFile(path, 1024); err == nil {
		t.Error("overwrote an existing image")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "existing disk data" {
		t.Fatalf("existing image was modified (size=%d, error=%v)", len(data), err)
	}
}

func TestCreateEmptyFileFailureLeavesNoImage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "disk.img")
	if err := (LinuxSystemAPI{}).CreateEmptyFile(path, -1); err == nil {
		t.Fatal("accepted negative image size")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("failed image remains: %v", err)
	}
}
