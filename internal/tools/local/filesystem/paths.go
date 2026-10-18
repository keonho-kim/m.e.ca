// Purpose: Resolve filesystem targets and select a trusted write root without lexical prefix checks.
// Pattern: Filesystem boundary parser.
// Usage: File adapters select os.Root handles; OS adapters consume the same canonical paths.
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
func (b *Boundary) WriteTarget(path string) (absolute, root string, err error) {
	absolute, err = b.ReadTarget(path)
	if err != nil {
		return "", "", err
	}
	for _, d := range b.writes {
		if Within(d.path, absolute) && len(d.path) > len(root) {
			root = d.path
		}
	}
	if root == "" {
		return "", "", fmt.Errorf("write outside allowed roots: %s", absolute)
	}
	if err := checkTarget(root, absolute); err != nil {
		return "", "", err
	}
	return absolute, root, nil
}

// Movable rejects renaming a root or an ancestor of protected state.
func (b *Boundary) Movable(path string) error {
	for _, root := range b.writes {
		if path == root.path {
			return fmt.Errorf("cannot move a write root: %s", path)
		}
	}
	for _, secret := range b.protected {
		if Within(path, secret) {
			return fmt.Errorf("cannot move protected path ancestor: %s", path)
		}
	}
	return nil
}
