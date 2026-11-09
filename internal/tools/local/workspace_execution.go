// Purpose: Return prepared workspace ownership on execution, rejection or abandoned preparation.
// Pattern: Execution resource decorator.
// Usage: prepare attaches one scoped hold; the runtime's existing Release contract returns it.
// Related: internal/tools/local/prepare.go, internal/agent/tool/release.go.
package local

import (
	"context"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type workspaceExecution struct {
	toolapi.Execution
	release func()
}

func (e *workspaceExecution) Release() { toolapi.Release(e.Execution); e.release() }
func (e *workspaceExecution) Execute(ctx context.Context, approved bool) (toolapi.Result, error) {
	defer e.Release()
	return e.Execution.Execute(ctx, approved)
}
