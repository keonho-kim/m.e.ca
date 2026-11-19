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
	return Replace(root, name, content)
}

// Replace publishes one complete private file relative to an already-owned root.
func Replace(root *os.Root, name string, content []byte) (err error) {
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
	return SyncDirectory(root)
}

// PublishMissing exposes only complete content and never replaces an existing file.
func PublishMissing(root *os.Root, name string, content []byte) error {
	added, err := publishMissing(root, name, func() ([]byte, error) { return content, nil })
	if added {
		err = errors.Join(err, SyncDirectory(root))
	}
	return err
}

// PublishMissingFiles publishes files in one directory and flushes its entries
// once. directoryChanged records child directories already created by the caller.
func PublishMissingFiles(root *os.Root, names []string, read func(string) ([]byte, error), directoryChanged bool) (err error) {
	defer func() {
		if directoryChanged {
			err = errors.Join(err, SyncDirectory(root))
		}
	}()
	for _, name := range names {
		added, err := publishMissing(root, name, func() ([]byte, error) { return read(name) })
		directoryChanged = directoryChanged || added
		if err != nil {
			return err
		}
	}
	return nil
}

func publishMissing(root *os.Root, name string, read func() ([]byte, error)) (added bool, err error) {
	if _, err := root.Lstat(name); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	content, err := read()
	if err != nil {
		return false, err
	}
	temporary := ".bootstrap-" + rand.Text()
	file, err := root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return false, err
	}
	defer func() {
		removeErr := root.Remove(temporary)
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}()
	_, writeErr := file.Write(content)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	if err = errors.Join(writeErr, file.Close()); err != nil {
		return false, err
	}
	if err = root.Link(temporary, name); errors.Is(err, os.ErrExist) {
		return false, nil
	}
	return err == nil, err
}

// SyncDirectory commits changes to the entries of an already-owned directory.
func SyncDirectory(root *os.Root) error {
	directory, err := root.Open(".")
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}
