package backup

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/elecbug/linuxus/src/internal/common/ruleset"
)

// The archive contains exactly disk.img and manifest.json; archive paths are
// never used as host paths. A manifest records identity, ownership and checksum.
type Manifest struct {
	Version int       `json:"version"`
	UserID  string    `json:"user_id"`
	Size    int64     `json:"size"`
	SHA256  string    `json:"sha256"`
	UID     int       `json:"uid"`
	GID     int       `json:"gid"`
	Created time.Time `json:"created"`
}

func Write(output io.Writer, image *os.File, manifest Manifest) error {
	info, err := image.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() <= 1024*1024 || !ruleset.AllowedUserID(manifest.UserID) {
		return fmt.Errorf("invalid disk image or user")
	}
	if _, err := image.Seek(0, io.SeekStart); err != nil {
		return err
	}
	manifest.Version = 1
	manifest.Size = info.Size()
	manifest.Created = time.Now().UTC()
	gz := gzip.NewWriter(output)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "disk.img", Typeflag: tar.TypeReg, Mode: 0600, Size: manifest.Size}); err != nil {
		return err
	}
	hash := sha256.New()
	if n, err := io.Copy(io.MultiWriter(tw, hash), image); err != nil {
		return fmt.Errorf("copy disk image: %w", err)
	} else if n != manifest.Size {
		return fmt.Errorf("disk image size changed during backup")
	}
	manifest.SHA256 = hex.EncodeToString(hash.Sum(nil))
	data, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	if err := tw.WriteHeader(&tar.Header{Name: "manifest.json", Typeflag: tar.TypeReg, Mode: 0600, Size: int64(len(data))}); err != nil {
		return err
	}
	if _, err := tw.Write(data); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}

// Read verifies every archive entry and the full disk checksum before returning.
// Destination must be an unpublished temporary file (nil for verification).
func Read(input io.Reader, destination *os.File, maxSize int64) (Manifest, error) {
	var manifest Manifest
	gz, err := gzip.NewReader(input)
	if err != nil {
		return manifest, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	header, err := tr.Next()
	if err != nil {
		return manifest, err
	}
	if header.Name != "disk.img" || header.Typeflag != tar.TypeReg || header.Size <= 1024*1024 || (maxSize > 0 && header.Size > maxSize) {
		return manifest, fmt.Errorf("invalid disk image entry or insufficient restore space")
	}
	size := header.Size
	hash := sha256.New()
	var output io.Writer = hash
	if destination != nil {
		output = io.MultiWriter(hash, &sparseWriter{file: destination})
	}
	if _, err := io.CopyBuffer(output, tr, make([]byte, 64<<10)); err != nil {
		return manifest, err
	}
	header, err = tr.Next()
	if err != nil {
		return manifest, err
	}
	if header.Name != "manifest.json" || header.Typeflag != tar.TypeReg || header.Size > 16<<10 {
		return manifest, fmt.Errorf("invalid backup manifest")
	}
	data, err := io.ReadAll(tr)
	if err != nil {
		return manifest, err
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, err
	}
	if manifest.Version != 1 || !ruleset.AllowedUserID(manifest.UserID) || manifest.Size != size || manifest.SHA256 != hex.EncodeToString(hash.Sum(nil)) || manifest.UID <= 0 || manifest.GID <= 0 {
		return manifest, fmt.Errorf("backup identity, size, ownership or checksum mismatch")
	}
	if _, err := tr.Next(); err != io.EOF {
		return manifest, fmt.Errorf("unexpected extra backup entry")
	}
	// Reading through gzip's EOF also validates its checksum and rejects payload
	// hidden after the tar terminator, including additional gzip members.
	trailing, err := io.Copy(io.Discard, gz)
	if err != nil {
		return manifest, err
	}
	if trailing != 0 {
		return manifest, fmt.Errorf("trailing backup payload")
	}
	if destination != nil {
		if err := destination.Truncate(size); err != nil {
			return manifest, err
		}
		if err := destination.Sync(); err != nil {
			return manifest, err
		}
	}
	return manifest, nil
}

type sparseWriter struct{ file *os.File }

func (w *sparseWriter) Write(data []byte) (int, error) {
	zero := true
	for _, b := range data {
		if b != 0 {
			zero = false
			break
		}
	}
	if zero {
		_, err := w.file.Seek(int64(len(data)), io.SeekCurrent)
		if err != nil {
			return 0, err
		}
		return len(data), nil
	}
	return w.file.Write(data)
}
