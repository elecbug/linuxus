package system_api

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMountTableIncludesBindMountsAndEscapedPaths(t *testing.T) {
	data := []byte("25 1 8:1 / / rw - ext4 /dev/sda rw\n" +
		"26 25 8:1 /source /volumes/homes/alice rw - ext4 /dev/sda rw\n" +
		"27 26 8:1 /nested /volumes/homes/alice/a\\040b\\134c rw - ext4 /dev/sda rw\n" +
		"28 25 8:1 /other /volumes/homes-other/bob rw - ext4 /dev/sda rw\n" +
		"29 27 8:1 /stack /volumes/homes/alice/a\\040b\\134c rw - ext4 /dev/sda rw\n")
	points, err := parseMountPoints(data)
	if err != nil {
		t.Fatal(err)
	}
	got := mountPointsUnder("/volumes/homes", points)
	want := []string{"/volumes/homes/alice", `/volumes/homes/alice/a b\c`, `/volumes/homes/alice/a b\c`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mounts=%v, want %v", got, want)
	}
	if _, err := parseMountPoints([]byte("incomplete mount record\n")); err == nil {
		t.Fatal("ignored invalid mount table")
	}
}

func TestMountDetectionResolvesStorageLink(t *testing.T) {
	api := LinuxSystemAPI{}
	alias := filepath.Join(t.TempDir(), "storage")
	if err := os.Symlink("/proc", alias); err != nil {
		t.Fatal(err)
	}
	mounted, err := api.IsMountPoint(alias)
	if err != nil || !mounted {
		t.Fatalf("linked mount undetected: %v", err)
	}
	got, err := api.MountPointsUnder(alias)
	if err != nil {
		t.Fatal(err)
	}
	want, err := api.MountPointsUnder("/proc")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("linked storage mounts differ: %v != %v", got, want)
	}
	// A symlink to a directory on another filesystem is not necessarily a mount.
	ordinary := filepath.Join(t.TempDir(), "process")
	if err := os.Symlink("/proc/self", ordinary); err != nil {
		t.Fatal(err)
	}
	if mounted, err := api.IsMountPoint(ordinary); err != nil || mounted {
		t.Fatalf("ordinary linked directory treated as mount: %t %v", mounted, err)
	}
	if mounted, err := api.IsMountPoint(filepath.Join(t.TempDir(), "missing")); err != nil || mounted {
		t.Fatalf("missing mount=%t, error=%v", mounted, err)
	}
}
