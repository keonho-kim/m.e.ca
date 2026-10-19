// Purpose: Pair one actual outcome with its native call and update execution observations.
// Pattern: Ordered result application.
// Usage: Single and batch dispatch commit outcomes in model call order.
// Related: internal/agent/run/action.go, internal/agent/run/observations.go.
package run

import (
	"context"
	"encoding/json"
	"errors"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

func (s *execution) acceptResult(ctx context.Context, v validatedCall, outcome toolapi.Result, toolErr error) error {
	result, failed := string(outcome.Content), outcome.Failed
	if errors.Is(toolErr, errRuntimeRecord) {
		if outcome.Started {
			return errors.Join(toolErr, s.observeResult(v, outcome))
		}
		return toolErr
	}
	s.result.HadToolFailure = s.result.HadToolFailure || failed || toolErr != nil
	if ctx.Err() != nil {
		return errors.Join(ctx.Err(), s.observeResult(v, outcome))
	}
	if toolErr != nil || failed {
		failure := outcome.Failure
		if failure == nil {
			failure = toolapi.FailureOf(toolErr)
		}
		if failure == nil {
			failure = &toolapi.Failure{Code: toolapi.ExecutionFailed, Detail: "tool reported failure", Next: "Inspect its output before choosing another action."}
		}
		failure.Detail = s.runner.Redact(failure.Detail)
		encoded, encodeErr := json.Marshal(struct {
			Failure *toolapi.Failure
			Output  json.RawMessage
		}{failure, outcome.Content})
		if encodeErr != nil {
			return encodeErr
		}
		result = string(encoded)
	}

	if err := s.save(ctx, modelapi.Message{Role: "tool", Content: result, ToolCallID: v.call.ID, ToolName: v.call.Name}); err != nil {
		return err
	}
	if err := s.observeResult(v, outcome); err != nil {
		return err
	}
	if errors.Is(toolErr, errApprovalDenied) {
		return toolErr
	}
	s.checkpoint.Phase = "observation"
	s.checkpoint.Arguments = nil
	s.checkpoint.Observation = s.runner.Redact(outcome.Summary)
	if outcome.Started {
		s.checkpoint.ChangeRevision++
	}
	if outcome.Failure != nil {
		s.checkpoint.Blocker = s.runner.Redact(outcome.Failure.Detail)
	} else {
		s.checkpoint.Blocker = ""
	}
	return s.persist(ctx)
}
