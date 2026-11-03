// Purpose: Own one image target and revalidate its path before taking the raster snapshot.
// Pattern: Concrete prepared read execution.
// Usage: Executor.Prepare returns this operation for view_image.
// Related: internal/tools/local/image.go, internal/tools/local/operation.go.
package local

import (
	"context"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type imageOperation struct {
	operation
	source target
}

func (op *imageOperation) Execute(ctx context.Context, approved bool) (toolapi.Result, error) {
	if err := op.claim(ctx, approved); err != nil {
		return toolapi.Result{}, err
	}
	if _, err := op.currentTarget(op.source); err != nil {
		return toolapi.Result{}, err
	}
	return op.viewImage(ctx)
}
