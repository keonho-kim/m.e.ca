// Purpose: Prepare policy-controlled operations and bind approval previews to original file versions.
// Pattern: Approval preparation use case.
// Usage: The task runtime calls Executor.Prepare before requesting approval.
// Related: internal/tools/local/operation.go, internal/tools/local/paths.go, internal/tools/local/diff/diff.go.
package local

import (
	"context"
	"crypto/rand"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tools/local/textformat"
	"github.com/keonho-kim/meca/internal/tools/workspacegate"
)

func (executor *Executor) Prepare(ctx context.Context, settings policy.Settings, call toolapi.Call) (toolapi.Execution, error) {
	return executor.prepare(ctx, settings, call, nil)
}

// PrepareAdmitted binds command reservations before an executable candidate escapes.
func (executor *Executor) PrepareAdmitted(ctx context.Context, settings policy.Settings, call toolapi.Call, admit func(context.Context) (func(), error)) (toolapi.Execution, error) {
	if call.Name != "execute_command" || admit == nil {
		return nil, faults.Errorf(diagcatalog.LocalCommandAdmissionRequiresAnExecuteCommandCandidate)
	}
	return executor.prepare(ctx, settings, call, admit)
}

func (executor *Executor) prepare(ctx context.Context, settings policy.Settings, call toolapi.Call, admit func(context.Context) (func(), error)) (result toolapi.Execution, resultErr error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	spec, input, err := executor.parsedCall(call)
	if err != nil {
		return nil, err
	}
	if err := settings.Check(); err != nil {
		return nil, err
	}
	if spec.Name == "execute_command" || spec.Effect == policy.Change {
		release, err := workspacegate.Acquire(ctx)
		if err != nil {
			return nil, err
		}
		defer func() {
			if resultErr != nil {
				release()
			} else {
				result = &workspaceExecution{Execution: result, release: release}
			}
		}()
	}
	texts := []string{input.Content, input.Command}
	for _, edit := range input.Edits {
		texts = append(texts, edit.OldText, edit.NewText)
	}
	for _, text := range texts {
		if textformat.RedactSource(text, executor.redact) != text {
			return nil, faults.Errorf(diagcatalog.LocalToolArgumentsContainAProtectedCredential)
		}
	}
	initialize := func(op *operation) {
		op.executor, op.settings, op.spec, op.input, op.callID = executor, settings, spec, input, call.ID
		op.executionID, op.batchID = call.ExecutionID, call.BatchID
		op.approval = toolapi.Approval{ID: "operation_" + rand.Text(), Name: spec.Label}
	}
	if spec.Effect == policy.Change && executor.ActiveCommand() != "" {
		return nil, faults.Errorf(diagcatalog.LocalCommandsAreStillRunningWaitCommandOr)
	}
	switch spec.Name {
	case "list_skills", "load_skill", "read_skill_file":
		op := &skillOperation{}
		initialize(&op.operation)
		return op, nil
	case "execute_command":
		op := &commandOperation{admit: admit}
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
	case "view_image":
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
		return nil, faults.Errorf(diagcatalog.LocalUnknownPreparedTool)
	}
}
