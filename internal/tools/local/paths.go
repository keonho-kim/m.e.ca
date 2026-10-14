// Purpose: Resolve real filesystem targets within the selected Scope and protected-state boundary.
// Pattern: Filesystem boundary.
// Usage: Preparation, execution and reference selection resolve paths here.
// Related: internal/tools/local/executor.go, internal/tools/local/version.go, internal/policy/path.go.
package local

import (
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/policy"
	"os"
	"path/filepath"
	"strings"
)

type target struct {
	absolute, relative string
	root               *os.Root
}

func realPath(path string) (string, error) {
	var tail []string
	for {
		resolved, err := filepath.EvalSymlinks(path)
		if err == nil {
			for index := len(tail) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, tail[index])
			}
			return resolved, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if info, statErr := os.Lstat(path); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
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

func (executor *Executor) resolve(scope policy.Scope, path string) (target, error) {
	absolute, _, err := policy.Resolve(executor.project, executor.home, scope, path)
	if err != nil {
		return target{}, err
	}
	absolute, err = realPath(absolute)
	if err != nil {
		return target{}, err
	}
	if absolute == executor.private || strings.HasPrefix(absolute, executor.private+string(filepath.Separator)) {
		return target{}, fmt.Errorf("access to V credentials and state is denied")
	}
	_, relative, err := policy.Resolve(executor.project, executor.home, scope, absolute)
	if err != nil {
		return target{}, err
	}
	root := executor.projectRoot
	if scope == policy.System {
		root = executor.systemRoot
	}
	return target{absolute: absolute, relative: relative, root: root}, nil
}
