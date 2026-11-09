// Purpose: Bind approval and one-shot admission shared by concrete prepared tool operations.
// Pattern: One-shot operation lifecycle.
// Usage: Concrete file, directory, command, control and image executions claim their own gate.
// Related: internal/tools/local/prepare.go, internal/tools/local/textedit/edit.go.
package local

import (
	"context"

	"sync/atomic"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/policy"
	toolcatalog "github.com/keonho-kim/meca/internal/tools/local/catalog"
)

type operation struct {
	callID   string
	executor *Executor
	settings policy.Settings
	spec     toolcatalog.Specification
	input    toolcatalog.Arguments
	approval toolapi.Approval
	executed atomic.Bool
}

func (op *operation) Approval() toolapi.Approval { return op.approval }

func (op *operation) claim(ctx context.Context, approved bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := op.settings.Check(); err != nil {
		return err
	}
	if (op.approval.Required || op.approval.UserRequired) && !approved {
		return faults.Errorf(diagcatalog.LocalOperationDeniedApprovalRequired)
	}
	if !op.executed.CompareAndSwap(false, true) {
		return faults.Errorf(diagcatalog.LocalPreparedOperationHasAlreadyExecuted)
	}
	return nil
}

func (op *operation) mutable() error {
	if op.executor.ActiveCommand() != "" {
		return faults.Errorf(diagcatalog.LocalCommandsAreStillRunningWaitCommandOr)
	}
	return nil
}

func (op *operation) currentTarget(original target) (target, error) {
	current, err := op.executor.resolve(op.settings.Scope, op.input.Path)
	if err != nil {
		return target{}, err
	}
	if current.absolute != original.absolute {
		return target{}, &faults.Failure{Code: diagcatalog.StaleTarget, Detail: diagcatalog.Text(diagcatalog.LocalTargetChangedAfterPreparation), Next: diagcatalog.Text(diagcatalog.LocalInspectTheIntendedPathAndItsCurrent)}
	}
	return current, nil
}
