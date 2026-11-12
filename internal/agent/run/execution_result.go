// Purpose: Commit actual outcomes, native replies and checkpoints before publishing them.
// Pattern: Unit of Work on the serial runtime owner.
// Usage: Workers return outcomes; single and batch owners call recordExecution serially.
// Related: internal/agent/run/execute.go, internal/agent/run/batch.go.
package run

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"

	"context"
	"encoding/json"
	"errors"
	"time"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

func (s *execution) recordExecution(ctx context.Context, v validatedCall, ready *readyExecution, outcome toolapi.Result, runErr error, emit func(runrecord.Event) error) (toolapi.Result, error) {
	recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	r, call, proposal := s.runner, ready.call, ready.proposal
	if runErr == nil && !outcome.Failed {
		if err := s.checkSkillResult(call, outcome); err != nil {
			// The file read had no side effect. Return an ordinary bounded tool
			// failure before committing an instruction we cannot retain.
			outcome = toolapi.Result{Failed: true, Content: json.RawMessage(`{}`), Failure: faults.FailureOf(err)}
			runErr = err
		}
	}
	event := runrecord.Event{CallID: call.ID, Call: &call}
	event.OccurredAt = time.Now().UTC().Format(time.RFC3339Nano)
	commands, ownsCommands := r.Tools.(toolapi.CommandExecutor)
	completedCommand := ownsCommands && outcome.Process != nil && outcome.Process.CommandID != "" && !toolapi.CommandActive(outcome.Process.Status)
	if completedCommand && len(outcome.Capture) == 0 && !commandOutputUnavailable(outcome) {
		capture, err := commands.CommandCapture(recordCtx, outcome.Process.CommandID)
		if err != nil {
			outcome, err = s.failedCommandCapture(outcome, outcome.Process.CommandID, err)
			if err != nil {
				return outcome, errors.Join(errRuntimeRecord, err)
			}
			runErr = outcome.Failure
		}
		outcome.Capture = capture
	}
	if v.definition.Effect == policy.Change {
		outcome.Started = true
	}
	outcome.Target = proposal.Target
	event.Kind, event.Text = "tool_succeeded", outcome.Summary
	if outcome.Failed {
		event.Kind = "tool_failed"
	}
	if runErr != nil {
		if outcome.Failure == nil {
			outcome.Failure = faults.FailureOf(runErr)
		}
		outcome.Failed = true
		event.Kind = "tool_failed"
	}
	if outcome.Failure != nil {
		event.ErrorCode = string(outcome.Failure.Code)
		// Version and instruction guards reject candidates before any mutation.
		if outcome.Failure.Code == diagcatalog.StaleTarget || outcome.Failure.Code == diagcatalog.InstructionChanged {
			outcome.Started = false
		}
		outcome.Failure.Detail = r.Redact(outcome.Failure.Detail)
		outcome.Failure.Next = r.Redact(outcome.Failure.Next)
		event.Text = outcome.Failure.Error()
	}
	opFact := toolapi.Operation{Target: outcome.Target, Versions: outcome.Versions, Started: outcome.Started, RunID: s.checkpoint.ID, CallID: call.ID, Name: call.Name, Status: event.Kind, Changes: outcome.Changes, Result: event.Text, Failure: outcome.Failure, Process: outcome.Process}
	opFact.Revision = s.checkpoint.ChangeRevision
	if call.Name == "execute_command" {
		opFact.Command = proposal.Target
	} else if outcome.Process != nil {
		opFact.Command = outcome.Process.Command
	}
	event.Operation = &opFact
	event.Source = outcome.Source
	event.Output = outcome.Content
	if outcome.Skill != nil && event.Kind == "tool_succeeded" {
		event.Skill = skillReadMetadata(*outcome.Skill)
	}
	event.Capture = outcome.Capture
	if event.Kind == "tool_succeeded" && proposal.Diff {
		event.ChangePreview = boundedChangePreview(r.Redact(proposal.Preview))
	}
	if err := s.acceptResult(recordCtx, v, outcome, runErr, event); err != nil {
		return outcome, errors.Join(errRuntimeRecord, err)
	}
	if completedCommand {
		if err := commands.AcknowledgeCommand(outcome.Process.CommandID); err != nil {
			return outcome, errors.Join(errRuntimeRecord, err)
		}
	}
	event.Capture = nil // Original captures are durable, never copied into live presentation events.
	if err := emit(event); err != nil && !(ctx.Err() != nil && errors.Is(err, ctx.Err())) {
		return outcome, errors.Join(errRuntimeRecord, err)
	}
	if completedCommand && commandOutputUnavailable(outcome) {
		return outcome, errors.Join(errRuntimeRecord, outcome.Failure)
	}
	return outcome, runErr
}
