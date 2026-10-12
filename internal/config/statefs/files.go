// Purpose: Publish complete private files atomically, either replacing or preserving an existing file.
// Pattern: Simple Module (filesystem boundary).
// Usage: Used by TOML persistence and first-run bootstrap.
// Related: internal/config/statefs/root.go, internal/config/config.go, internal/config/prompts/prompts.go.

package statefs

import (
	"crypto/rand"
	"errors"
	"os"
)

func Write(directory, name string, content []byte) (err error) {
	root, err := Open(directory)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	temporary := ".pending-" + rand.Text()
	file, err := root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
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
	if err = errors.Join(writeErr, file.Close()); err != nil {
		return err
	}
	if err = root.Rename(temporary, name); err != nil {
		return err
	}
	directoryFile, err := root.Open(".")
	if err != nil {
		return err
	}
	return errors.Join(directoryFile.Sync(), directoryFile.Close())
}

// PublishMissing exposes only complete content and never replaces an existing file.
func PublishMissing(root *os.Root, name string, content []byte) (err error) {
	temporary := ".bootstrap-" + rand.Text()
	file, err := root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() {
		removeErr := root.Remove(temporary)
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}()
	_, writeErr := file.Write(content)
	if err = errors.Join(writeErr, file.Sync(), file.Close()); err != nil {
		return err
	}
	if err = root.Link(temporary, name); errors.Is(err, os.ErrExist) {
		return nil
	}
	return err
}
