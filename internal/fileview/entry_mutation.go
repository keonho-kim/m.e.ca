// Purpose: Rename, move or remove one workspace entry under pinned directory handles.
// Pattern: Revision-checked filesystem write adapter.
// Usage: The server holds its workspace effect lease for MutateEntry.
// Related: internal/fileview/directory_snapshot.go, internal/commands/server/file_mutation.go.
package fileview

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/policy"
)

type EntryMutation struct {
	Path, Revision, Name, Folder string
	Remove                       bool
}

func validEntryName(name string) bool {
	return len(name) <= 255 && utf8.ValidString(name) && strings.TrimSpace(name) != "" &&
		name != "." && name != ".." && !strings.ContainsAny(name, "/\\\x00\r\n")
}

func (e *Reader) MutateEntry(ctx context.Context, settings policy.Settings, change EntryMutation, protect func(string) error) (updated string, resultErr error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := settings.Check(); err != nil {
		return "", err
	}
	name := filepath.Base(filepath.Clean(change.Path))
	if settings.Scope != policy.Directory || protect == nil || !validEntryName(name) || len(change.Revision) != 64 {
		return "", fault.New(diag.InvalidArgument, diag.UIFileMutationInvalid)
	}
	parent, err := e.resolve(settings.Scope, filepath.Dir(change.Path))
	if err != nil {
		return "", err
	}
	absolute := filepath.Join(parent.Absolute, name)
	if err := e.checkMutationPath(settings.Scope, absolute, protect); err != nil {
		return "", err
	}
	source, err := parent.Root.OpenRoot(parent.Relative)
	if err != nil {
		return "", err
	}
	defer func() { resultErr = errors.Join(resultErr, source.Close()) }()
	info, err := source.Lstat(name)
	if err != nil {
		return "", err
	}
	if !info.IsDir() && !info.Mode().IsRegular() {
		return "", fault.New(diag.InvalidArgument, diag.UIFileMutationInvalid)
	}
	if change.Remove {
		if err := e.checkMutationDirectory(ctx, settings.Scope, parent, source, change.Revision); err != nil {
			return "", err
		}
		if info.IsDir() {
			err = removeEntry(ctx, source, name)
		} else {
			err = source.Remove(name)
		}
		return "", err
	}

	destination, nextName := parent, name
	if change.Name != "" && validEntryName(change.Name) {
		nextName = change.Name
	} else if change.Folder != "" && change.Name == "" {
		destination, err = e.resolve(settings.Scope, change.Folder)
		if err != nil {
			return "", err
		}
	} else {
		return "", fault.New(diag.InvalidArgument, diag.UIFileMutationInvalid)
	}
	next := filepath.Join(destination.Absolute, nextName)
	if relative, err := filepath.Rel(absolute, destination.Absolute); err == nil && info.IsDir() && filepath.IsLocal(relative) {
		return "", fault.New(diag.InvalidArgument, diag.UIFileMutationInvalid)
	}
	if err := e.checkMutationPath(settings.Scope, next, protect); err != nil {
		return "", err
	}
	dest := source
	if destination.Absolute != parent.Absolute {
		dest, err = destination.Root.OpenRoot(destination.Relative)
		if err != nil {
			return "", err
		}
		defer func() { resultErr = errors.Join(resultErr, dest.Close()) }()
		if err := e.checkMutationDirectory(ctx, settings.Scope, destination, dest, ""); err != nil {
			return "", err
		}
	}
	if err := e.checkMutationDirectory(ctx, settings.Scope, parent, source, change.Revision); err != nil {
		return "", err
	}
	if absolute == next {
		return next, nil
	}
	applied, err := renameEntry(source, name, dest, nextName)
	if applied {
		return next, err
	}
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", fault.New(diag.Conflict, diag.UIFileMutationExists)
		}
		return "", err
	}
	return next, nil
}

func (e *Reader) checkMutationPath(scope policy.Scope, path string, protect func(string) error) error {
	target, err := e.resolve(scope, path)
	if err != nil {
		return err
	}
	if target.Absolute != path && !caseAlias(target, path) || e.redact(path) != path {
		return fault.New(diag.Conflict, diag.UIFileMutationChanged)
	}
	return protect(path)
}

// A physical path on case-insensitive volumes retains the existing spelling.
// Accept only the same non-symlink entry, leaving collision safety to the atomic rename.
func caseAlias(target Target, path string) bool {
	if filepath.Dir(target.Absolute) != filepath.Dir(path) ||
		!strings.EqualFold(filepath.Base(target.Absolute), filepath.Base(path)) {
		return false
	}
	actual, err := target.Root.Lstat(target.Relative)
	if err != nil || actual.Mode()&os.ModeSymlink != 0 {
		return false
	}
	requested, err := target.Root.Lstat(filepath.Join(filepath.Dir(target.Relative), filepath.Base(path)))
	return err == nil && requested.Mode()&os.ModeSymlink == 0 && os.SameFile(actual, requested)
}

func (e *Reader) checkMutationDirectory(ctx context.Context, scope policy.Scope, target Target, root *os.Root, revision string) error {
	pinned, err := root.Stat(".")
	if err != nil {
		return err
	}
	current, err := e.resolve(scope, target.Absolute)
	if err != nil {
		return err
	}
	actual, err := current.Root.Stat(current.Relative)
	if err != nil {
		return err
	}
	if current.Absolute != target.Absolute || !os.SameFile(actual, pinned) || revision != "" && directoryRevision(actual) != revision {
		return fault.New(diag.Conflict, diag.UIFileMutationChanged)
	}
	return ctx.Err()
}
