// Purpose: Resolve selected file and directory paths under tool access policy.
// Pattern: Filesystem Adapter.
// Usage: Called by file browsing and selected-path revalidation.
// Related: internal/tools/local/paths.go, internal/tools/local/references_test.go.
package local

import (
	"github.com/keonho-kim/v-code/internal/policy"
	"path/filepath"
	"strings"
)

type Reference struct {
	Path      string
	Directory bool
}

func (e *Executor) Reference(settings policy.Settings, path string) (Reference, error) {
	t, err := e.resolve(settings.Scope, path)
	if err != nil {
		return Reference{}, err
	}
	info, err := t.root.Stat(t.relative)
	if err != nil {
		return Reference{}, err
	}
	relative, err := filepath.Rel(e.project, t.absolute)
	if err != nil {
		return Reference{}, err
	}
	if relative == ".." || strings.HasPrefix(relative, "../") {
		relative = t.absolute
	}
	if info.IsDir() {
		relative += "/"
	}
	return Reference{Path: relative, Directory: info.IsDir()}, nil
}
