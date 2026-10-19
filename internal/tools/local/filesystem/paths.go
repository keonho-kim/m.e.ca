// Purpose: Canonicalize file paths and reject direct file-tool access to protected state.
// Pattern: Filesystem boundary parser.
// Usage: Local file adapters resolve targets before approval and version checks.
// Related: internal/tools/local/filesystem/boundary.go.
package filesystem

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Within(parent, path string) bool {
	relative, err := filepath.Rel(parent, path)
	return err == nil && (relative == "." || filepath.IsLocal(relative))
}
func Canonical(path string) (string, error) {
	if !filepath.IsAbs(path) || strings.ContainsRune(path, 0) {
		return "", fmt.Errorf("expected absolute filesystem path")
	}
	path = filepath.Clean(path)
	var tail []string
	for {
		resolved, err := filepath.EvalSymlinks(path)
		if err == nil {
			resolved, err = physicalPath(resolved)
			if err != nil {
				return "", err
			}
			for i := len(tail) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, tail[i])
			}
			return resolved, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if info, e := os.Lstat(path); e == nil && info.Mode()&os.ModeSymlink != 0 {
			return "", err
		}
		parent := filepath.Dir(path)
		if parent == path {
			return "", err
		}
		tail = append(tail, filepath.Base(path))
		path = parent
	}
}
func (b *Boundary) ReadTarget(path string) (string, error) {
	absolute, err := Canonical(path)
	if err != nil {
		return "", err
	}
	for _, secret := range b.protected {
		if Within(secret, absolute) {
			return "", fmt.Errorf("protected path: %s", absolute)
		}
	}
	return absolute, nil
}
