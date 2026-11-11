//go:build darwin || linux

// Purpose: Bind both parent directories before atomically renaming an entry.
// Pattern: Descriptor-owned native filesystem adapter.
// Usage: MutateEntry calls this after its path and revision checks.
// Related: internal/fileview/entry_mutation.go.
package fileview

import (
	"errors"
	"os"
)

func renameEntry(source *os.Root, name string, destination *os.Root, next string) (applied bool, resultErr error) {
	from, err := source.Open(".")
	if err != nil {
		return false, err
	}
	defer func() { resultErr = errors.Join(resultErr, from.Close()) }()
	to := from
	if destination != source {
		to, err = destination.Open(".")
		if err != nil {
			return false, err
		}
		defer func() { resultErr = errors.Join(resultErr, to.Close()) }()
	}
	err = renameNoReplace(int(from.Fd()), name, int(to.Fd()), next)
	return err == nil, err
}
