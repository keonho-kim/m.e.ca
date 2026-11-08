// Purpose: Apply shared-workspace admission around real local effects without bypassing approval.
// Pattern: Prepared operation decorator.
// Usage: Files reserve before preparation; commands reserve known targets only at execution.
// Related: internal/tools/team/executor.go, internal/tools/team/workspace/workspace.go.
package team

import (
	"context"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/tools/team/workspace"
)

type localOperation struct {
	executor  *Executor
	call      toolapi.Call
	operation toolapi.Execution
	held      *workspace.Lease
	paths     []string
	denialKey string
}

func (op *localOperation) Approval() toolapi.Approval { return op.operation.Approval() }
func (op *localOperation) Release() {
	op.held.Release()
	op.held = nil
	toolapi.Release(op.operation)
}
func (op *localOperation) Execute(ctx context.Context, approved bool) (toolapi.Result, error) {
	defer op.Release()
	e := op.executor
	if e.Refusals.Refused(op.operation.Approval(), &op.denialKey) {
		return toolapi.Result{}, faults.Errorf(diagcatalog.TeamTheUserRefusedThisOperationDoNot)
	}
	switch op.call.Name {
	case "wait_command", "cancel_command", "list_command_runs", "list_skills", "load_skill", "view_image", "read_skill_file":
		return op.operation.Execute(ctx, approved)
	}

	if err := e.Peer.CheckUserDecision(); err != nil {
		return toolapi.Result{Failed: true, Failure: faults.FailureOf(err)}, err
	}
	if err := e.Peer.CheckWork(op.paths); err != nil {
		failure := &faults.Failure{Code: diagcatalog.StaleTarget, Detail: err.Error(), Next: diagcatalog.Text(diagcatalog.TeamReadCurrentTeammateWorkAndRegenerateThis)}
		return toolapi.Result{Failed: true, Failure: failure}, failure
	}
	result, err := op.operation.Execute(ctx, approved)
	if op.call.Name == "execute_command" && result.Process != nil && len(op.paths) > 0 {
		e.trackCommandPaths(result.Process.CommandID, op.paths)
	}

	return result, err
}
