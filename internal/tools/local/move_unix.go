//go:build linux || darwin

// Purpose: Move entries across allowed roots using pinned parents and exclusive destination creation.
// Pattern: Filesystem Adapter.
// Usage: Approved move_file operations call moveTargets after version checks.
// Related: internal/tools/local/mutations.go, internal/tools/local/move_darwin.go, internal/tools/local/move_linux.go.
package local

import (
	"errors"
	"path/filepath"
)

func moveTargets(source, destination target) (err error) {
	from, err := source.root.Open(filepath.Dir(source.relative))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, from.Close()) }()
	to, err := destination.root.Open(filepath.Dir(destination.relative))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, to.Close()) }()
	return renameExclusive(int(from.Fd()), filepath.Base(source.relative), int(to.Fd()), filepath.Base(destination.relative))
}
