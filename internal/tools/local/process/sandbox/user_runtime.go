// Purpose: Expose explicit user-installed runtime directories without granting access to the whole home or arbitrary PATH entries.
// Pattern: Runtime filesystem and environment preparation.
// Usage: Platform launch preparation calls userRuntime before constructing its read-only rules.
// Related: internal/tools/local/process/sandbox/linux.go, internal/tools/local/process/sandbox/darwin.go.
package sandbox

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func userRuntime(launch *Launch, home, private string) ([]string, error) {
	home, err := filepath.EvalSymlinks(home)
	if err != nil {
		return nil, err
	}
	bin := filepath.Join(home, ".local", "bin")
	python := filepath.Join(home, ".local", "share", "uv", "python")
	var paths []string
	for _, path := range []string{bin, python} {
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return nil, err
		}
		// Do not turn an installation-directory symlink into a broad home or
		// private-state exception. Links within it still obey the sandbox view.
		if !info.IsDir() || resolved != path {
			return nil, fmt.Errorf("user runtime directory must not redirect through symlinks: %s", path)
		}
		if containsPath(private, path) || containsPath(path, private) {
			return nil, fmt.Errorf("user runtime overlaps V state: %s", path)
		}
		paths = append(paths, path)
		if path == bin {
			for i, value := range launch.Environment {
				if strings.HasPrefix(value, "PATH=") && !slices.Contains(filepath.SplitList(strings.TrimPrefix(value, "PATH=")), bin) {
					launch.Environment[i] = value + string(os.PathListSeparator) + bin
				}
			}
		} else {
			launch.Environment = append(launch.Environment, "UV_PYTHON_INSTALL_DIR="+python)
		}
	}
	return paths, nil
}

func containsPath(parent, path string) bool {
	relative, err := filepath.Rel(parent, path)
	return err == nil && filepath.IsLocal(relative)
}
