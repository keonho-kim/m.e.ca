// Purpose: Own frozen recursive directory targets and guard their creation lifecycle.
// Pattern: Concrete prepared directory execution.
// Usage: Executor.Prepare creates this operation for create_directory.
// Related: internal/tools/local/directories.go, internal/tools/local/directory_execution.go.
package local

import (
	"context"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type directoryOperation struct {
	operation
	directories    []directoryTarget
	directoryPaths []string
}

func (op *directoryOperation) checkMutationSafety() error {
	for _, dir := range op.directories {
		if dir.info == nil {
			if err := op.checkMutationPath(dir.target.absolute); err != nil {
				return err
			}
		}
	}
	return nil
}

func (op *directoryOperation) Execute(ctx context.Context, approved bool) (toolapi.Result, error) {
	if err := op.claim(ctx, approved); err != nil {
		return toolapi.Result{}, err
	}
	if err := op.mutable(); err != nil {
		return toolapi.Result{}, err
	}
	if err := op.checkMutationSafety(); err != nil {
		return toolapi.Result{}, err
	}
	return op.createDirectories(ctx)
}
