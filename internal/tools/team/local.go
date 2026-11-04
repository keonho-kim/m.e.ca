// Purpose: Apply shared-workspace admission around real local effects without bypassing approval.
// Pattern: Prepared operation decorator.
// Usage: Executor.Prepare wraps local operations; command completion releases transferred ownership.
// Related: internal/tools/team/executor.go, internal/tools/team/workspace/workspace.go.
package team

import (
	"context"
	"errors"
	"fmt"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/team/workspace"
)

type localOperation struct {
	executor  *Executor
	call      toolapi.Call
	operation toolapi.Execution
	held      *workspace.Lease
	paths     []string
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
	if e.Refusals.Refused(op.operation.Approval()) {
		return toolapi.Result{}, fmt.Errorf("the user refused this operation; do not repeat it through another peer")
	}
	switch op.call.Name {
	case "wait_command", "cancel_command", "list_command_runs":
		result, err := op.operation.Execute(ctx, approved)
		e.releaseFinishedShell()
		return result, err
	case "view_image":
		select {
		case e.Images <- struct{}{}:
		case <-ctx.Done():
			return toolapi.Result{}, ctx.Err()
		}
		defer func() { <-e.Images }()
		return op.operation.Execute(ctx, approved)
	}
	if err := e.Peer.CheckUserDecision(); err != nil {
		return toolapi.Result{Failed: true, Failure: toolapi.FailureOf(err)}, err
	}
	if err := e.Peer.CheckWork(op.paths); err != nil {
		failure := &toolapi.Failure{Code: toolapi.StaleTarget, Detail: err.Error(), Next: "Read current teammate work and regenerate this operation after coordinating the relevant path."}
		return toolapi.Result{Failed: true, Failure: failure}, failure
	}
	result, err := op.operation.Execute(ctx, approved)
	if op.call.Name == "execute_command" && result.Process != nil && len(op.paths) > 0 {
		e.trackCommandPaths(result.Process.CommandID, op.paths)
	}
	if op.call.Name == "execute_command" && result.Process != nil && result.Process.Status == "running" {
		held := op.held
		op.held = nil
		err = errors.Join(err, e.holdShell(result.Process.CommandID, held))
	}
	return result, err
}
