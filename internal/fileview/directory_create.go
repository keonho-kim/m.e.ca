// Purpose: Create one explicitly named directory under a freshly checked, pinned parent.
// Pattern: Guarded filesystem write adapter.
// Usage: Server folder creation calls this while holding its shared workspace effect lease.
// Related: internal/commands/server/folder.go, internal/fileview/reader.go, internal/tools/local/paths.go.
package fileview

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/policy"
)

func (e *Reader) CreateDirectory(ctx context.Context, settings policy.Settings, parent, name string) (created string, resultErr error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := settings.Check(); err != nil {
		return "", err
	}
	if !validEntryName(name) {
		return "", fault.New(diag.InvalidArgument, diag.UIFolderNameInvalid)
	}
	target, err := e.resolve(settings.Scope, parent)
	if err != nil {
		return "", err
	}
	child, err := e.resolve(settings.Scope, filepath.Join(target.Absolute, name))
	if err != nil {
		return "", err
	}
	if child.Absolute != filepath.Join(target.Absolute, name) || e.redact(child.Absolute) != child.Absolute {
		return "", fault.New(diag.Conflict, diag.UIFolderChanged)
	}
	directory, err := target.Root.OpenRoot(target.Relative)
	if err != nil {
		return "", err
	}
	defer func() { resultErr = errors.Join(resultErr, directory.Close()) }()
	pinned, err := directory.Stat(".")
	if err != nil {
		return "", err
	}
	current, err := e.resolve(settings.Scope, parent)
	if err != nil {
		return "", err
	}
	info, err := current.Root.Stat(current.Relative)
	if err != nil {
		return "", err
	}
	if current.Absolute != target.Absolute || !os.SameFile(pinned, info) {
		return "", fault.New(diag.Conflict, diag.UIFolderChanged)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := directory.Mkdir(name, 0700); err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", fault.New(diag.Conflict, diag.UIFolderExists)
		}
		return "", err
	}
	handle, err := directory.Open(".")
	if err != nil {
		return "", err
	}
	if err := errors.Join(handle.Sync(), handle.Close()); err != nil {
		return "", err
	}
	createdTarget, err := e.resolve(settings.Scope, child.Absolute)
	if err != nil {
		return "", err
	}
	return createdTarget.Absolute, nil
}
