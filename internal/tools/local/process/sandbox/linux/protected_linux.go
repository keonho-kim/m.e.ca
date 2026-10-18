//go:build linux

// Purpose: Mask protected host paths in the otherwise read-only filesystem projection.
// Pattern: Linux mount Adapter.
// Usage: enterRoot applies masks after mounting writable roots and before pivot_root.
// Related: internal/tools/local/process/sandbox/linux/mounts_linux.go.
package linux

import (
	"errors"
	"os"
	"path/filepath"
)

func hideProtected(root, path, scratch string) (err error) {
	info, err := os.Lstat(filepath.Join(root, path))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return hidePrivate(root, path)
	}
	file, err := os.CreateTemp(scratch, "masked-")
	if err != nil {
		return err
	}
	name := file.Name()
	if err := errors.Join(file.Chmod(0000), file.Close()); err != nil {
		return err
	}
	defer func() { err = errors.Join(err, os.Remove(name)) }()
	return bind(root, binding{path: path, source: name, readOnly: true})
}
