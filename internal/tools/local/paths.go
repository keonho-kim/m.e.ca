// Purpose: Resolve canonical file-tool targets against project or host roots and private-path guards.
// Pattern: Filesystem boundary.
// Usage: Preparation, execution and reference selection resolve paths here.
// Related: internal/tools/local/executor.go, internal/tools/local/version.go, internal/policy/path.go.
package local

import (
	"os"

	"github.com/keonho-kim/meca/internal/policy"
)

type target struct {
	absolute, relative string
	root               *os.Root
}

func (executor *Executor) resolve(scope policy.Scope, path string) (target, error) {
	if err := executor.boundary.Check(); err != nil {
		return target{}, err
	}
	return executor.resolveTarget(scope, path)
}

// Callers sharing this path within a planning phase must check the pinned
// project before and after the phase. Execution always uses fresh resolve.
func (executor *Executor) resolveTarget(scope policy.Scope, path string) (target, error) {
	absolute, _, err := policy.Resolve(executor.project, executor.home, scope, path)
	if err != nil {
		return target{}, err
	}
	absolute, err = executor.boundary.ReadTarget(absolute)
	if err != nil {
		return target{}, err
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
