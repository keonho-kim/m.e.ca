// Purpose: Start an approved owned command and return its first timed observation.
// Pattern: Tool Adapter.
// Usage: Called by the one-shot operation after required safety approval.
// Related: internal/tools/local/commands.go, internal/tools/local/command_archive.go.
package local

import (
	"context"
	"errors"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func (op *commandOperation) command(ctx context.Context) (toolapi.Result, error) {
	var id string
	var err error
	if op.admit == nil {
		if _, err = op.admitCommand(ctx); err != nil {
			return toolapi.Result{}, err
		}
		id, err = op.executor.commands.Start(ctx, op.executor.project, op.input.Command, op.environment, op.callID)
	} else {
		id, err = op.executor.commands.StartAdmitted(ctx, op.executor.project, op.input.Command, op.environment, op.callID, op.admitCommand)
	}
	if err != nil {
		if id != "" {
			state, waitErr := op.executor.commands.Wait(context.Background(), id, 0)
			result, readErr := op.executor.commandResult(state)
			return result, errors.Join(err, waitErr, readErr)
		}
		return toolapi.Result{}, err
	}
	state, err := op.executor.commands.Wait(ctx, id, op.executor.commandWait)
	if err != nil {
		state, cancelErr := op.executor.commands.Cancel(context.Background(), id)
		result, readErr := op.executor.commandResult(state)
		return result, errors.Join(err, cancelErr, readErr)
	}
	return op.executor.commandResult(state)
}
