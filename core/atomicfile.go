package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data to a temporary sibling and renames it into place.
// A failed write leaves an existing destination untouched.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	return WriteAtomic(path, perm, func(w io.Writer) error {
		_, err := w.Write(data)
		return err
	})
}

// WriteAtomic streams a file to a temporary sibling and renames it into place.
func WriteAtomic(path string, perm os.FileMode, write func(io.Writer) error) (err error) {
	if path == "" {
		return fmt.Errorf("atomic write: output path is empty")
	}
	if write == nil {
		return fmt.Errorf("atomic write: writer callback is nil")
	}
	dir := filepath.Dir(path)
	file, err := os.CreateTemp(dir, ".pdf-tmp-*")
	if err != nil {
		return err
	}
	tmp := file.Name()
	defer func() {
		_ = file.Close()
		_ = os.Remove(tmp)
	}()
	if err = file.Chmod(perm); err != nil {
		return err
	}
	if err = write(file); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
