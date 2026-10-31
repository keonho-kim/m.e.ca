// Purpose: Start an approved owned command and return its first timed observation.
// Pattern: Tool Adapter.
// Usage: Called by the one-shot operation after required safety approval.
// Related: internal/tools/local/commands.go, internal/tools/local/command_archive.go.
package local

import (
	"context"
	"errors"
	"fmt"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func (op *operation) command(ctx context.Context) (toolapi.Result, error) {
	id, err := op.executor.commands.Start(ctx, op.executor.project, op.input.Command, op.environment, op.callID)
	if err != nil {
		return toolapi.Result{}, err
	}
	state, err := op.executor.commands.Wait(ctx, id, op.executor.commandWait)
	if err != nil {
		_, _ = op.executor.commands.Cancel(context.Background(), id)
		result, readErr := op.executor.readCommandResult(id)
		return result, errors.Join(err, readErr)
	}
	if state.Error != "" {
		return toolapi.Result{}, fmt.Errorf("command result archive failed for %s: %s", id, state.Error)
	}
	if state.Status == "running" {
		return op.executor.runningCommand(state)
	}
	return op.executor.readCommandResult(id)
}
