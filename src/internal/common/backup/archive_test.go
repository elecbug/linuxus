package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func testArchive(t *testing.T) ([]byte, []byte) {
	t.Helper()
	data := make([]byte, 2<<20)
	copy(data, "student files")
	data[1080] = 0x53
	data[1081] = 0xef
	copy(data[len(data)-100:], "last block")
	path := filepath.Join(t.TempDir(), "disk.img")
	os.WriteFile(path, data, 0600)
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var output bytes.Buffer
	if err := Write(&output, file, Manifest{UserID: "alice", UID: 1000, GID: 1000}); err != nil {
		t.Fatal(err)
	}
	return output.Bytes(), data
}

func TestArchiveRoundTripAndSparseRestore(t *testing.T) {
	encoded, want := testArchive(t)
	manifest, err := Read(bytes.NewReader(encoded), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.UserID != "alice" {
		t.Fatal(manifest)
	}
	path := filepath.Join(t.TempDir(), "restored.img")
	output, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Read(bytes.NewReader(encoded), output, 4<<20); err != nil {
		t.Fatal(err)
	}
	output.Close()
	got, _ := os.ReadFile(path)
	if !bytes.Equal(got, want) {
		t.Fatal("restored bytes differ")
	}
	if _, err := Read(bytes.NewReader(encoded), nil, 1<<20); err == nil {
		t.Fatal("ignored restore space limit")
	}
}

func TestRejectTruncatedCorruptAndUnexpectedEntries(t *testing.T) {
	encoded, _ := testArchive(t)
	if _, err := Read(bytes.NewReader(encoded[:len(encoded)-5]), nil, 0); err == nil {
		t.Fatal("accepted truncated backup")
	}
	reader, _ := gzip.NewReader(bytes.NewReader(encoded))
	raw, _ := io.ReadAll(reader)
	reader.Close()
	raw[512] ^= 0xff
	var corrupt bytes.Buffer
	gz := gzip.NewWriter(&corrupt)
	gz.Write(raw)
	gz.Close()
	if _, err := Read(bytes.NewReader(corrupt.Bytes()), nil, 0); err == nil {
		t.Fatal("accepted checksum mismatch")
	}
	for _, name := range []string{"../../outside", "manifest.json", "disk.img"} {
		var output bytes.Buffer
		gz := gzip.NewWriter(&output)
		tw := tar.NewWriter(gz)
		tw.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeSymlink, Linkname: "/etc/shadow"})
		tw.Close()
		gz.Close()
		if _, err := Read(bytes.NewReader(output.Bytes()), nil, 0); err == nil {
			t.Fatal("accepted unsafe archive entry")
		}
	}
	if _, err := Read(bytes.NewReader(append(append([]byte{}, encoded...), encoded...)), nil, 0); err == nil {
		t.Fatal("accepted extra compressed member")
	}
}
