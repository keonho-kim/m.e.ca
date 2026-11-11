// Purpose: Delete directory trees in bounded batches with cancellation and pinned child identities.
// Pattern: Context-aware filesystem traversal.
// Usage: Reader.MutateEntry owns the enclosing effect lease and parent revision checks.
// Related: internal/fileview/entry_mutation.go, internal/fileview/entry_remove_test.go.
package fileview

import (
	"context"
	"errors"
	"io"
	"os"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func removeEntry(ctx context.Context, parent *os.Root, name string) (resultErr error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	info, err := parent.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return parent.Remove(name)
	}
	child, err := parent.OpenRoot(name)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, child.Close()) }()
	pinned, err := child.Stat(".")
	if err != nil {
		return err
	}
	if !os.SameFile(info, pinned) {
		return fault.New(diag.Conflict, diag.UIFileMutationChanged)
	}
	entries, err := child.Open(".")
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, entries.Close()) }()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		names, readErr := entries.Readdirnames(64)
		for _, name := range names {
			if err := removeEntry(ctx, child, name); err != nil {
				return err
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	current, err := parent.Lstat(name)
	if err != nil {
		return err
	}
	if !os.SameFile(info, current) {
		return fault.New(diag.Conflict, diag.UIFileMutationChanged)
	}
	return parent.Remove(name)
}
