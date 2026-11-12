// Purpose: Preserve asynchronous command completions and resolve outstanding commands before reporting.
// Pattern: Serial resource observation boundary.
// Usage: The decision loop collects results; request shutdown cancels and joins owned commands.
// Related: internal/agent/tool/commands.go, internal/agent/run/run.go.
package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

func (s *execution) collectCommands(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	commands, ok := s.runner.Tools.(toolapi.CommandExecutor)
	if !ok {
		return false, nil
	}
	completed, err := commands.CollectCommands()
	if err != nil {
		return false, errors.Join(errRuntimeRecord, err)
	}
	var terminal error
	for _, finished := range completed {
		result := finished.Result
		if !commandOutputUnavailable(result) {
			result.Capture, err = commands.CommandCapture(ctx, finished.ID)
			if err != nil {
				result, err = s.failedCommandCapture(result, finished.ID, err)
				if err != nil {
					return false, errors.Join(errRuntimeRecord, err)
				}
			}
		}
		status := "tool_succeeded"
		if result.Failed {
			status = "tool_failed"
		}
		arguments, _ := json.Marshal(map[string]string{"command": finished.Command})
		call := toolapi.Call{ID: finished.CallID, ExecutionID: finished.ExecutionID, BatchID: finished.BatchID, Name: "execute_command", Arguments: arguments}
		op := toolapi.Operation{RunID: s.checkpoint.ID, CallID: finished.CallID, Name: "execute_command", Command: finished.Command, Started: result.Started, Status: status, Process: result.Process, Failure: result.Failure, Result: result.Summary, Revision: s.checkpoint.ChangeRevision}
		s.checkpoint.Work.Observe(op)
		s.checkpoint.Observation = result.Summary
		s.result.HadToolFailure = s.result.HadToolFailure || result.Failed
		event := runrecord.Event{Kind: "command_finished", CallID: call.ID, Call: &call, Operation: &op, Output: result.Content, Capture: result.Capture, Text: result.Summary}
		checkpoint, err := s.checkpointEvent()
		if err != nil {
			return false, err
		}
		events := []runrecord.Event{event, checkpoint}
		// Keep the original JSON identical to the outcome payload. Context
		// projection supplies the observation label without duplicating storage.
		message := modelapi.Message{Role: "system", Content: string(result.Content), ToolCallID: call.ID, ToolName: call.Name}
		if err := s.save(ctx, message, events...); err != nil {
			if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
				return false, ctx.Err()
			}
			return false, err
		}
		if err := commands.AcknowledgeCommand(finished.ID); err != nil {
			return false, errors.Join(errRuntimeRecord, err)
		}
		s.commandRevision++
		event.Capture = nil
		if err := s.emit(event); err != nil && !errors.Is(err, context.Canceled) && ctx.Err() == nil {
			return false, err
		}
		if commandOutputUnavailable(result) {
			terminal = errors.Join(terminal, errRuntimeRecord, result.Failure)
		}
	}
	return len(completed) > 0, terminal
}

func commandOutputUnavailable(result toolapi.Result) bool {
	return result.Failure != nil && diag.Semantic(result.Failure.Code) == diag.PersistenceFailed
}

func (s *execution) pendingCommands(ctx context.Context) (bool, error) {
	changed, err := s.collectCommands(ctx)
	if err != nil || changed {
		return changed, err
	}
	commands, ok := s.runner.Tools.(toolapi.CommandExecutor)
	if !ok || !commands.HasCommands() {
		return false, nil
	}
	pending := commands.CommandActivity()
	if len(pending) == 0 {
		return false, nil
	}
	body, _ := json.Marshal(pending)
	message := fmt.Sprintf("Commands from this request are still active: %s\n%s\nObserve their terminal results before the final answer.", body, toolapi.CommandManagementHint)
	return true, s.save(ctx, modelapi.Message{Role: "system", Content: message})
}

func (s *execution) stopCommands(ctx context.Context) error {
	commands, ok := s.runner.Tools.(toolapi.CommandExecutor)
	if !ok {
		return nil
	}
	commands.StopCommands()
	flush, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	_, err := s.collectCommands(flush)
	return err
}
