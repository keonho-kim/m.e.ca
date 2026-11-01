// Purpose: Execute native list, wait and cancel controls for executor-owned commands.
// Pattern: Concrete prepared command-control adapter.
// Usage: Executor.Prepare returns these controls without reinterpreting them as shell commands.
// Related: internal/tools/local/commands.go, internal/tools/local/process/registry.go.
package local

import (
	"context"
	"fmt"
	"time"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/local/process"
	resultfmt "github.com/keonho-kim/meca/internal/tools/result"
)

type commandControlOperation struct{ operation }

func (op *commandControlOperation) Execute(ctx context.Context, approved bool) (toolapi.Result, error) {
	if err := op.claim(ctx, approved); err != nil {
		return toolapi.Result{}, err
	}
	return op.commandControl(ctx)
}

func (op *commandControlOperation) commandControl(ctx context.Context) (toolapi.Result, error) {
	e := op.executor
	if op.spec.Name == "list_command_runs" {
		runs := e.CommandActivity()
		return resultfmt.Encode(struct {
			Runs []toolapi.CommandRun "json:\"runs\""
		}{runs}, fmt.Sprintf("%d owned command runs", len(runs)))
	}
	duration := process.DefaultWait
	if op.input.WaitSeconds != nil {
		duration = time.Duration(*op.input.WaitSeconds) * time.Second
	}
	var state process.State
	var err error
	if op.spec.Name == "cancel_command" {
		state, err = e.commands.Cancel(ctx, op.input.CommandID)
	} else {
		state, err = e.commands.Wait(ctx, op.input.CommandID, duration)
	}
	if err != nil {
		// Collected commands are no longer active; their executor-owned archive
		// still supports repeated inspection without signaling an OS PID.
		if result, readErr := e.readCommandResult(op.input.CommandID); readErr == nil {
			if op.spec.Name == "cancel_command" {
				result.Failed = false
				result.Failure = nil
			}
			return result, nil
		}
		return toolapi.Result{}, err
	}
	if state.Status == "running" {
		return e.runningCommand(state)
	}
	if state.Error != "" {
		return toolapi.Result{}, fmt.Errorf("command result archive failed for %s: %s", state.ID, state.Error)
	}
	result, err := e.readCommandResult(state.ID)
	if result.Process != nil {
		result.Process.Command = e.redact(state.Command)
	}
	if op.spec.Name == "cancel_command" && err == nil {
		result.Failed = false
		result.Failure = nil
	}
	return result, err
}
