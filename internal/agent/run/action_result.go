// Purpose: Pair one actual outcome with its native call and update execution observations.
// Pattern: Ordered result application.
// Usage: Single and batch dispatch commit outcomes in model call order.
// Related: internal/agent/run/action.go, internal/agent/run/observations.go.
package run

import (
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"

	"context"
	"crypto/rand"
	"encoding/json"
	"errors"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

func (s *execution) acceptResult(ctx context.Context, v validatedCall, outcome toolapi.Result, toolErr error, events ...runrecord.Event) error {
	result, failed := string(outcome.Content), outcome.Failed
	if errors.Is(toolErr, errRuntimeRecord) {
		if outcome.Started {
			return errors.Join(toolErr, s.observeResult(v, outcome))
		}
		return toolErr
	}
	if outcome.Depart && toolErr == nil && !failed {
		s.result.Departed = true
	}
	s.result.HadToolFailure = s.result.HadToolFailure || failed || toolErr != nil
	if ctx.Err() != nil {
		return errors.Join(ctx.Err(), s.observeResult(v, outcome))
	}
	if toolErr != nil || failed {
		failure := outcome.Failure
		if failure == nil {
			failure = faults.FailureOf(toolErr)
		}
		if failure == nil {
			failure = &faults.Failure{Code: diagcatalog.ExecutionFailed, Detail: diagcatalog.Text(diagcatalog.RunToolReportedFailure), Next: diagcatalog.Text(diagcatalog.RunInspectItsOutputBeforeChoosingAnotherAction)}
		}
		failure.Detail = s.runner.Redact(failure.Detail)
		encoded, encodeErr := json.Marshal(struct {
			Failure *faults.Failure
			Output  json.RawMessage
		}{failure, outcome.Content})
		if encodeErr != nil {
			return encodeErr
		}
		result = string(encoded)
	}

	if err := s.observeResult(v, outcome); err != nil {
		return err
	}
	if errors.Is(toolErr, errApprovalDenied) {
		return s.save(ctx, modelapi.Message{Source: outcome.Source, Role: "tool", Content: result, Images: outcome.Images, ToolCallID: v.call.ID, ToolName: v.call.Name}, events...)
	}
	s.checkpoint.Phase = "observation"
	s.checkpoint.Observation = s.runner.Redact(outcome.Summary)
	if outcome.Started {
		s.checkpoint.ChangeRevision++
	}
	if outcome.Failure != nil {
		s.checkpoint.Blocker = s.runner.Redact(outcome.Failure.Detail)
	} else {
		s.checkpoint.Blocker = ""
	}
	event, err := s.checkpointEvent()
	if err != nil {
		return err
	}
	events = append(events, event)
	message := modelapi.Message{Source: outcome.Source, Role: "tool", Content: result, Images: outcome.Images, ToolCallID: v.call.ID, ToolName: v.call.Name}
	if (outcome.Skill != nil || outcome.SkillFile != nil) && !failed && toolErr == nil {
		message.ID = "skill_" + rand.Text()
	}
	if err := s.save(ctx, message, events...); err != nil {
		return err
	}
	if outcome.Skill != nil && !failed && toolErr == nil {
		if err := s.retainSkillRead(*outcome.Skill, s.messages[len(s.messages)-1]); err != nil {
			return err
		}
	}
	if outcome.SkillFile != nil && !failed && toolErr == nil {
		if err := s.retainSkillReference(*outcome.SkillFile, s.messages[len(s.messages)-1]); err != nil {
			return err
		}
	}
	return nil
}
