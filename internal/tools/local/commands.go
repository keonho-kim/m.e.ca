// Purpose: Adapt owned command lifecycle and process controls to native tool results.
// Pattern: Session lifecycle adapter.
// Usage: Runtime binds a request; prepared control operations inspect only this executor.
// Related: internal/tools/local/command_archive.go, internal/tools/local/process/registry.go.
package local

import (
	"context"
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
	e.liveOutputs.clear()
	e.commands.Begin(ctx, deadline)
}
func (e *Executor) StopCommands()                   { e.commands.End(); e.liveOutputs.clear() }
func (e *Executor) CommandUpdates() <-chan struct{} { return e.commands.Updates() }
func (e *Executor) activeCommands() bool {
	for _, state := range e.commands.List() {
		if state.Status == "running" {
			return true
		}
	}
	return false
}
func (e *Executor) CommandActivity() []toolapi.CommandRun {
	out := make([]toolapi.CommandRun, 0)
	for _, state := range e.commands.List() {
		run := toolapi.CommandRun{CallID: state.CallID, ID: state.ID, Command: e.redact(state.Command), Status: state.Status, ElapsedMillis: state.ElapsedMillis, LastOutputMillis: state.LastOutputMillis}
		if state.Status != "running" {
			code := state.ExitCode
			run.ExitCode = &code
		}
		out = append(out, run)
	}
	return out
}

func (e *Executor) CollectCommands() ([]toolapi.CommandCompletion, error) {
	var completed []toolapi.CommandCompletion
	for _, state := range e.commands.List() {
		if state.Status == "running" {
			continue
		}
		if state.Error != "" {
			return nil, fmt.Errorf("command result archive failed: %s", state.Error)
		}
		result, err := e.readCommandResult(state.ID)
		if err != nil {
			return nil, err
		}
		result.Process.Command = e.redact(state.Command)
		completed = append(completed, toolapi.CommandCompletion{ID: state.ID, CallID: state.CallID, Command: e.redact(state.Command), Result: result})
	}
	return completed, nil
}

func (e *Executor) runningCommand(state process.State) (toolapi.Result, error) {
	captures, err := e.commandCaptures(state.ID)
	if err != nil {
		return toolapi.Result{}, err
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
	}{state.ID, "running", stdout, stderr, stdoutPath, stderrPath, state.ElapsedMillis, state.LastOutputMillis, toolapi.CommandManagementHint}
	result, err := resultfmt.Encode(body, "Command is still running: "+state.ID)
	result.Started = true
	result.Process = &toolapi.ProcessOutcome{CommandID: state.ID, Command: e.redact(state.Command), Status: "running"}
	return result, err
}

func (op *operation) commandControl(ctx context.Context) (toolapi.Result, error) {
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
