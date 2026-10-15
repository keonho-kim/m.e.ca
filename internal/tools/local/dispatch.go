// Purpose: Dispatch prepared operations to file and command adapters.
// Pattern: Tool execution dispatcher.
// Usage: Operation.Execute dispatches after all policy and version rechecks.
// Related: internal/tools/local/operation.go, internal/tools/local/file/read.go, internal/tools/local/mutations.go.
package local

import (
	"context"
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	fileops "github.com/keonho-kim/v-code/internal/tools/local/file"
)

func (op *operation) run(ctx context.Context) (toolapi.Result, error) {
	switch op.spec.name {
	case "read_file":
		return (fileops.Read{Root: op.source.root, Path: op.source.relative, Absolute: op.source.absolute, Offset: op.input.Offset, Limit: op.input.Limit, Anchors: &op.executor.reads, Redact: op.executor.redact}).Execute(ctx)
	case "write_file", "edit_file", "delete_file", "move_file":
		return op.mutate()
	case "execute_command":
		return op.command(ctx)
	default:
		return toolapi.Result{}, fmt.Errorf("unknown prepared tool")
	}
}
