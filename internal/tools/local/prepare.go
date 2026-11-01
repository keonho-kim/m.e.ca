// Purpose: Prepare policy-controlled operations and bind approval previews to original file versions.
// Pattern: Approval preparation use case.
// Usage: The task runtime calls Executor.Prepare before requesting approval.
// Related: internal/tools/local/operation.go, internal/tools/local/paths.go, internal/tools/local/diff/diff.go.
package local

import (
	"context"
	"crypto/rand"
	"fmt"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	toolcatalog "github.com/keonho-kim/meca/internal/tools/local/catalog"
	"github.com/keonho-kim/meca/internal/tools/local/textformat"
)

func (executor *Executor) Prepare(ctx context.Context, settings policy.Settings, call toolapi.Call) (toolapi.Execution, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	spec, input, err := toolcatalog.Parse(call)
	if err != nil {
		return nil, err
	}
	if err := settings.Check(); err != nil {
		return nil, err
	}
	texts := []string{input.Content, input.Command}
	for _, edit := range input.Edits {
		texts = append(texts, edit.OldText, edit.NewText)
	}
	for _, text := range texts {
		if textformat.RedactSource(text, executor.redact) != text {
			return nil, fmt.Errorf("tool arguments contain a protected credential")
		}
	}
	initialize := func(op *operation) {
		op.executor, op.settings, op.spec, op.input, op.callID = executor, settings, spec, input, call.ID
		op.approval = toolapi.Approval{ID: "operation_" + rand.Text(), Name: spec.Label}
	}
	if spec.Effect == policy.Change && executor.activeCommands() {
		return nil, fmt.Errorf("commands are still running; wait_command or cancel_command before changing files")
	}
	switch spec.Name {
	case "execute_command":
		op := &commandOperation{}
		initialize(&op.operation)
		if err := op.prepareCommand(ctx); err != nil {
			return nil, err
		}
		return op, nil
	case "list_command_runs", "wait_command", "cancel_command":
		op := &commandControlOperation{}
		initialize(&op.operation)
		op.approval.Target = input.CommandID
		return op, nil
	case "create_directory":
		op := &directoryOperation{}
		initialize(&op.operation)
		if err := op.prepareDirectories(call); err != nil {
			return nil, err
		}
		return op, op.checkMutationSafety()
	case "read_image":
		op := &imageOperation{}
		initialize(&op.operation)
		op.source, err = executor.resolve(settings.Scope, input.Path)
		if err != nil {
			return nil, err
		}
		op.approval.Target = op.source.absolute
		return op, nil
	case "write_file", "edit_file":
		op := &fileOperation{}
		initialize(&op.operation)
		op.source, err = executor.resolve(settings.Scope, input.Path)
		if err != nil {
			return nil, err
		}
		op.approval.Target = op.source.absolute
		if err := op.checkMutationPath(op.source.absolute); err != nil {
			return nil, err
		}
		if err := op.prepareChange(call); err != nil {
			return nil, err
		}
		return op, op.checkMutationPath(op.source.absolute)
	default:
		return nil, fmt.Errorf("unknown prepared tool")
	}
}
