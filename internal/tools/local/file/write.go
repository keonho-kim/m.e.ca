// Purpose: Atomically publish prepared file bytes using a temporary sibling file.
// Pattern: Filesystem write adapter.
// Usage: Approved local mutations call Write with the original permission mode.
// Related: internal/tools/local/mutations.go.
package file

import (
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
)

func Write(root *os.Root, path string, content []byte, mode os.FileMode) (err error) {
	parent := filepath.Dir(path)
	if err := root.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	temporary := filepath.Join(parent, ".meca-write-"+rand.Text())
	file, err := root.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := root.Remove(temporary); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}()
	_, writeErr := file.Write(content)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return err
	}
	return root.Rename(temporary, path)
}
