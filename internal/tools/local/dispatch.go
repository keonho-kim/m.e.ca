// Purpose: Dispatch prepared operations to file, search and process adapters.
// Pattern: Tool execution dispatcher.
// Usage: Operation.Execute dispatches after all policy and version rechecks.
// Related: internal/tools/local/operation.go, internal/tools/local/file/read.go, internal/tools/local/mutations.go.
package local

import (
	"context"
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	fileops "github.com/keonho-kim/v-code/internal/tools/local/file"
	"github.com/keonho-kim/v-code/internal/tools/local/search"
	resultfmt "github.com/keonho-kim/v-code/internal/tools/result"
)

func (op *operation) run(ctx context.Context) (toolapi.Result, error) {
	switch op.spec.name {
	case "read_file":
		return (fileops.Read{Root: op.source.root, Path: op.source.relative, Absolute: op.source.absolute, Offset: op.input.Offset, Limit: op.input.Limit, Anchors: &op.executor.reads, Redact: op.executor.redact}).Execute(ctx)
	case "list_directory":
		limit := op.input.Limit
		if limit == 0 {
			limit = 200
		}
		entries, more, err := op.executor.Directory(ctx, op.settings, op.source.absolute, op.input.Offset, limit)
		if err != nil {
			return toolapi.Result{}, err
		}
		return resultfmt.Encode(struct {
			Entries    []Entry `json:"entries"`
			More       bool    `json:"more"`
			NextOffset int     `json:"next_offset"`
		}{entries, more, op.input.Offset + limit}, fmt.Sprintf("%d entries", len(entries)))
	case "search_files", "grep":
		return (search.Request{Root: op.source.root, Path: op.source.relative, Private: op.executor.private, Pattern: op.input.Pattern, Content: op.spec.name == "grep", Redact: op.executor.redact}).Execute(ctx)
	case "create_directory", "write_file", "edit_file", "delete_file", "move_file":
		return op.mutate()
	case "execute_command":
		return op.command(ctx)
	default:
		return toolapi.Result{}, fmt.Errorf("unknown prepared tool")
	}
}
