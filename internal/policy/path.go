// Purpose: Resolve user paths against a fixed project root and lexical Scope.
// Pattern: Pure Function.
// Usage: File adapters combine these results with os.Root containment checks.
// Related: internal/policy/policy.go, internal/policy/policy_test.go, internal/tools/local.

package policy

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Resolve does not follow symlinks; the filesystem boundary must use os.Root.
func Resolve(project, home string, scope Scope, path string) (string, string, error) {
	if _, err := ParseScope(string(scope)); err != nil {
		return "", "", err
	}
	if !filepath.IsAbs(project) || !filepath.IsAbs(home) {
		return "", "", fmt.Errorf("project and home roots must be absolute")
	}
	if len(path) > 4096 || !utf8.ValidString(path) || strings.ContainsRune(path, 0) {
		return "", "", fmt.Errorf("invalid path")
	}
	if path == "~" {
		path = home
	} else if strings.HasPrefix(path, "~/") {
		path = filepath.Join(home, path[2:])
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(project, path)
	}
	path = filepath.Clean(path)
	root := project
	if scope == System {
		root = filepath.VolumeName(path) + string(filepath.Separator)
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("access denied by scope: %s", path)
	}
	return path, relative, nil
}
