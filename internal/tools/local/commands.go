// Purpose: Adapt owned command lifecycle and process controls to native tool results.
// Pattern: Session lifecycle adapter.
// Usage: Runtime binds a request; prepared control operations inspect only this executor.
// Related: internal/tools/local/command_archive.go, internal/tools/local/process/registry.go.
package local

import (
	"context"
	"errors"
	"fmt"
	"time"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/local/process"
	resultfmt "github.com/keonho-kim/meca/internal/tools/result"
)

func commandControl(name string) bool {
	return name == "list_command_runs" || name == "wait_command" || name == "cancel_command"
}
func (e *Executor) BeginCommands(ctx context.Context, deadline time.Time) {
	e.commands.Begin(ctx, deadline)
	e.liveOutputs.clear()
}
func (e *Executor) StopCommands() {
	e.liveOutputs.releaseProjections()
	e.commands.End()
	e.liveOutputs.clear()
}
func (e *Executor) CommandUpdates() <-chan struct{} { return e.commands.Updates() }
func (e *Executor) ActiveCommand() string           { return e.commands.ActiveID() }
func (e *Executor) HasCommands() bool               { return e.commands.HasCommands() }
func (e *Executor) CommandActivity() []toolapi.CommandRun {
	var storage [process.MaxActive]process.State
	states := e.commands.AppendStates(storage[:0])
	if len(states) == 0 {
		return nil
	}
	out := make([]toolapi.CommandRun, 0, len(states))
	for _, state := range states {
		run := toolapi.CommandRun{CallID: state.CallID, ID: state.ID, Command: e.commandLabel(state.ID, state.Command), Status: state.Status, ElapsedMillis: state.ElapsedMillis, LastOutputMillis: state.LastOutputMillis}
		if !toolapi.CommandActive(state.Status) {
			code := state.ExitCode
			run.ExitCode = &code
		}
		out = append(out, run)
	}
	return out
}

func (e *Executor) CollectCommands() ([]toolapi.CommandCompletion, error) {
	var storage [process.MaxActive]process.Completion
	states := e.commands.AppendCompleted(storage[:0])
	if len(states) == 0 {
		return nil, nil
	}
	completed := make([]toolapi.CommandCompletion, 0, len(states))
	for _, state := range states {
		if state.Error != "" {
			return nil, fmt.Errorf("command result archive failed: %s", state.Error)
		}
		result, err := e.readCommandResult(state.ID)
		if err != nil {
			return nil, err
		}
		command := e.commandLabel(state.ID, state.Command)
		result.Process.Command = command
		completed = append(completed, toolapi.CommandCompletion{ID: state.ID, CallID: state.CallID, Command: command, Result: result})
	}
	return completed, nil
}

func (e *Executor) runningCommand(state process.State) (toolapi.Result, error) {
	// A projection failure does not terminate the owned process. Preserve its
	// identity so callers retain mutation leases and can wait or cancel it.
	metadata := toolapi.Result{Started: state.Status != "waiting", Process: &toolapi.ProcessOutcome{CommandID: state.ID, Command: e.commandLabel(state.ID, state.Command), Status: state.Status}}
	captures, err := e.commandCaptures(state.ID)
	pending := errors.Is(err, process.ErrProjectionBusy)
	if err != nil && !pending {
		return metadata, err
	}
	stdout, stdoutPath := captures[0].excerpt, captures[0].path
	stderr, stderrPath := captures[1].excerpt, captures[1].path
	body := struct {
		CommandID        string "json:\"command_id\""
		Status           string "json:\"Status\""
		Stdout           string
		Stderr           string
		StdoutPath       string
		StderrPath       string
		ElapsedMillis    int64  "json:\"elapsed_millis\""
		LastOutputMillis *int64 "json:\"last_output_millis\""
		Next             string "json:\"next\""
		OutputPending    bool   `json:"output_pending,omitempty"`
	}{state.ID, state.Status, stdout, stderr, stdoutPath, stderrPath, state.ElapsedMillis, state.LastOutputMillis, toolapi.CommandManagementHint, pending}
	result, err := resultfmt.Encode(body, "Command "+state.Status+": "+state.ID)
	result.Started, result.Process = metadata.Started, metadata.Process
	return result, err
}
