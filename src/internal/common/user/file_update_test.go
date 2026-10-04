package user

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type faultyAuthFile struct {
	*os.File
	fail            string
	failure         error
	rollbackFailure error
}

func (f *faultyAuthFile) WriteAt(data []byte, offset int64) (int, error) {
	if f.fail == "write" || f.fail == "short" {
		step := f.fail
		f.fail = ""
		n, err := f.File.WriteAt(data[:len(data)/2], offset)
		if err != nil {
			return n, err
		}
		if step == "short" {
			return n, nil
		}
		return n, f.failure
	}
	return f.File.WriteAt(data, offset)
}
func (f *faultyAuthFile) Truncate(size int64) error {
	if f.fail == "truncate" {
		f.fail = ""
		return f.failure
	}
	if f.rollbackFailure != nil {
		return f.rollbackFailure
	}
	return f.File.Truncate(size)
}
func (f *faultyAuthFile) Sync() error {
	if f.fail == "sync" {
		f.fail = ""
		return f.failure
	}
	return f.File.Sync()
}

func TestCredentialUpdatesRestoreFileAfterPartialFailure(t *testing.T) {
	for _, operation := range []string{"append", "remove"} {
		for _, step := range []string{"write", "short", "truncate", "sync", "success"} {
			if operation == "append" && step == "truncate" {
				continue
			}
			t.Run(operation+"/"+step, func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "AUTH_LIST")
				original := []byte("alice:alice-hash\nbob:bob-hash\n")
				if err := os.WriteFile(path, original, 0600); err != nil {
					t.Fatal(err)
				}
				file, err := os.OpenFile(path, os.O_RDWR, 0600)
				if err != nil {
					t.Fatal(err)
				}
				defer file.Close()
				before, err := file.Stat()
				if err != nil {
					t.Fatal(err)
				}
				failure := errors.New("injected I/O failure")
				faulty := &faultyAuthFile{File: file, fail: step, failure: failure}
				updated := []byte("bob:bob-hash\n")
				if operation == "append" {
					entry := []byte("carol:carol-hash\n")
					updated = append(append([]byte{}, original...), entry...)
					err = appendAuthEntry(faulty, int64(len(original)), entry)
				} else {
					err = replaceAuthData(faulty, original, updated)
				}
				want := original
				if step == "success" {
					want = updated
					if err != nil {
						t.Fatal(err)
					}
				} else {
					expectedErr := failure
					if step == "short" {
						expectedErr = io.ErrShortWrite
					}
					if !errors.Is(err, expectedErr) {
						t.Fatalf("lost update failure: %v", err)
					}
				}
				got, readErr := os.ReadFile(path)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if string(got) != string(want) {
					t.Fatalf("credentials changed after %s failure: %q", step, got)
				}
				after, statErr := os.Stat(path)
				if statErr != nil {
					t.Fatal(statErr)
				}
				if !os.SameFile(before, after) {
					t.Fatal("bind mount inode changed")
				}
			})
		}
	}
}

func TestCredentialRollbackFailureIsReported(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "auth")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	original, recovery := errors.New("write failure"), errors.New("recovery failure")
	faulty := &faultyAuthFile{File: file, fail: "write", failure: original, rollbackFailure: recovery}
	err = appendAuthEntry(faulty, 0, []byte("alice:hash\n"))
	if !errors.Is(err, original) || !errors.Is(err, recovery) || !strings.Contains(err.Error(), "restore auth file") {
		t.Fatalf("lost recovery failure: %v", err)
	}
}
