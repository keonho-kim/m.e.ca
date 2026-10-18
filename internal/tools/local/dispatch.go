// Purpose: Dispatch prepared operations to file and command adapters.
// Pattern: Tool execution dispatcher.
// Usage: Operation.Execute dispatches after all policy and version rechecks.
// Related: internal/tools/local/operation.go, internal/tools/local/textedit/edit.go, internal/tools/local/mutations.go.
package local

import (
	"context"
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

func (op *operation) run(ctx context.Context) (toolapi.Result, error) {
	switch op.spec.name {
	case "write_file", "edit_file", "move_file":
		return op.mutate()
	case "execute_command":
		return op.command(ctx)
	default:
		return toolapi.Result{}, fmt.Errorf("unknown prepared tool")
	}
}
