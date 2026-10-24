// Purpose: Resolve shell paths without erasing symlink traversal or unlink semantics.
// Pattern: Filesystem path adapter.
// Usage: Command safety resolves targets without shell expansion or directory scans.
// Related: internal/tools/local/command_safety.go, internal/tools/local/filesystem/paths.go.
package local

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/keonho-kim/meca/internal/tools/local/filesystem"
)

func shellPath(path string, removal bool) (string, error) {
	trimmed := strings.TrimRight(path, "/")
	i := strings.LastIndex(trimmed, "/")
	base := trimmed[i+1:]
	if removal && !strings.HasSuffix(path, "/") && base != "." && base != ".." {
		parent, err := shellCanonical(path[:i+1])
		if err != nil {
			return "", err
		}
		return filepath.Join(parent, base), nil
	}
	return shellCanonical(path)
}

func shellCanonical(path string) (string, error) {
	// Clean/Join before EvalSymlinks would change `link/../target` semantics.
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return filesystem.Canonical(resolved)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	path = strings.TrimRight(path, "/")
	i := strings.LastIndex(path, "/")
	if i < 0 {
		return "", err
	}
	if info, e := os.Lstat(path); e == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", err
	}
	parent, e := shellCanonical(path[:i+1])
	if e != nil {
		return "", e
	}
	return filepath.Join(parent, path[i+1:]), nil
}
