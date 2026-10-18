// Purpose: Pair one actual outcome with its native call and update execution observations.
// Pattern: Ordered result application.
// Usage: Single and batch dispatch commit outcomes in model call order.
// Related: internal/agent/run/action.go, internal/agent/run/observation.go.
package run

import (
	"context"
	"encoding/json"
	"errors"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	"time"
)

func (s *execution) acceptResult(ctx context.Context, v validatedCall, outcome toolapi.Result, toolErr error) error {
	signature := callSignature(v.call)
	mutating := v.definition.Effect == policy.Change || v.definition.Effect == policy.Command
	result, failed := string(outcome.Content), outcome.Failed
	if errors.Is(toolErr, errRuntimeRecord) {
		if outcome.Started {
			return errors.Join(toolErr, s.observeCompletion(ctx, v, outcome, toolErr))
		}
		return toolErr
	}
	s.result.HadToolFailure = s.result.HadToolFailure || failed || toolErr != nil
	if ctx.Err() != nil {
		recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		return errors.Join(ctx.Err(), s.observeCompletion(recordCtx, v, outcome, toolErr))
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
	if toolErr == nil && !outcome.Failed && outcome.Started {
		s.textCorrection = false
	}
	if err := s.observeCompletion(ctx, v, outcome, toolErr); err != nil {
		return err
	}
	if errors.Is(toolErr, errApprovalDenied) {
		return toolErr
	}
	return s.observeOutcome(ctx, v.call, v.call.Arguments, signature, mutating, v.definition.Effect, result, outcome, toolErr)
}
