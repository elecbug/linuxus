package user

import (
	"errors"
	"fmt"
	"io"
)

// The caller holds the file's exclusive lock for both the update and recovery.
// Keep the inode: replacing the file would leave Docker reading an old bind mount.
type authFileEditor interface {
	WriteAt([]byte, int64) (int, error)
	Truncate(int64) error
	Sync() error
}

func writeAuthAt(file authFileEditor, data []byte, offset int64) error {
	n, err := file.WriteAt(data, offset)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	if err != nil {
		return fmt.Errorf("write auth file: %w", err)
	}
	return nil
}

func resizeAndSyncAuth(file authFileEditor, size int64) error {
	if err := file.Truncate(size); err != nil {
		return fmt.Errorf("truncate auth file: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync auth file: %w", err)
	}
	return nil
}

func appendAuthEntry(file authFileEditor, originalSize int64, entry []byte) error {
	err := writeAuthAt(file, entry, originalSize)
	if err == nil {
		if syncErr := file.Sync(); syncErr != nil {
			err = fmt.Errorf("sync auth file: %w", syncErr)
		}
	}
	if err != nil {
		if rollbackErr := resizeAndSyncAuth(file, originalSize); rollbackErr != nil {
			return errors.Join(err, fmt.Errorf("restore auth file: %w", rollbackErr))
		}
	}
	return err
}

func replaceAuthData(file authFileEditor, original, updated []byte) error {
	replace := func(data []byte) error {
		if err := writeAuthAt(file, data, 0); err != nil {
			return err
		}
		return resizeAndSyncAuth(file, int64(len(data)))
	}
	if err := replace(updated); err != nil {
		if rollbackErr := replace(original); rollbackErr != nil {
			return errors.Join(err, fmt.Errorf("restore auth file: %w", rollbackErr))
		}
		return err
	}
	return nil
}
