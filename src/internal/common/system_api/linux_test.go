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
