package util

import (
	"errors"
	"io"
	"os"
)

// MoveFile moves a file with cross-device fallback.
func MoveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := CopyFile(src, dst); err != nil {
		return err
	}
	return os.Remove(src)
}

// CopyFile copies a file to destination and fsyncs output.
func CopyFile(src, dst string) (retErr error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := in.Close(); closeErr != nil {
			if retErr == nil {
				retErr = closeErr
			} else {
				retErr = errors.Join(retErr, closeErr)
			}
		}
	}()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := out.Close(); closeErr != nil {
			if retErr == nil {
				retErr = closeErr
			} else {
				retErr = errors.Join(retErr, closeErr)
			}
		}
	}()

	if _, err := io.Copy(out, in); err != nil {
		retErr = err
		return
	}
	if err := out.Sync(); err != nil {
		retErr = err
		return
	}
	return
}
