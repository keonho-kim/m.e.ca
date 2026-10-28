// Purpose: Resolve selected file and directory paths under tool access policy.
// Pattern: Filesystem Adapter.
// Usage: Called by file browsing and selected-path revalidation.
// Related: internal/fileview/reader.go, internal/tools/local/references_test.go.
package fileview

import (
	"context"
	"github.com/keonho-kim/meca/internal/policy"
	"path/filepath"
	"strings"
)

type Reference struct {
	Path      string
	Directory bool
}

func (e *Reader) Reference(ctx context.Context, settings policy.Settings, path string) (Reference, error) {
	if err := ctx.Err(); err != nil {
		return Reference{}, err
	}
	t, err := e.resolve(settings.Scope, path)
	if err != nil {
		return Reference{}, err
	}
	info, err := t.Root.Stat(t.Relative)
	if err != nil {
		return Reference{}, err
	}
	relative, err := filepath.Rel(e.project, t.Absolute)
	if err != nil {
		return Reference{}, err
	}
	if relative == ".." || strings.HasPrefix(relative, "../") {
		relative = t.Absolute
	}
	if info.IsDir() {
		relative += "/"
	}
	return Reference{Path: relative, Directory: info.IsDir()}, nil
}
