// Purpose: Dispatch prepared operations to file and command adapters.
// Pattern: Tool execution dispatcher.
// Usage: Operation.Execute dispatches after all policy and version rechecks.
// Related: internal/tools/local/operation.go, internal/tools/local/textedit/edit.go, internal/tools/local/mutations.go.
package local

import (
	"context"
	"fmt"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func (op *operation) run(ctx context.Context) (toolapi.Result, error) {
	switch op.spec.Name {
	case "read_image":
		return op.readImage(ctx)
	case "write_file", "edit_file":
		return op.mutate()
	case "execute_command":
		return op.command(ctx)
	case "list_command_runs", "wait_command", "cancel_command":
		return op.commandControl(ctx)
	default:
		return toolapi.Result{}, fmt.Errorf("unknown prepared tool")
	}
}
