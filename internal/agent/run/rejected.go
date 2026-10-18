// Purpose: Record rejected preparation once and pair skipped siblings with concise cause references.
// Pattern: Ordered failure transition.
// Usage: Single and batch dispatch share this non-executing result boundary.
// Related: internal/agent/run/execute.go, internal/agent/run/batch_wave.go.
package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

func (s *execution) recordRejected(ctx context.Context, call toolapi.Call, outcome toolapi.Result, cause error, emit func(Event) error) error {
	failure := outcome.Failure
	if failure == nil {
		failure = toolapi.FailureOf(cause)
	}
	if failure == nil {
		return fmt.Errorf("rejected call requires its cause")
	}
	copy := *failure
	copy.Detail = s.runner.Redact(copy.Detail)
	copy.Next = s.runner.Redact(copy.Next)
	failure = &copy
	body, err := json.Marshal(struct {
		Executed bool             `json:"executed"`
		Failure  *toolapi.Failure `json:"failure"`
	}{false, failure})
	if err != nil {
		return err
	}
	e := Event{Kind: "tool_failed", CallID: call.ID, Call: &call, Text: s.runner.Redact(failure.Error()), Output: body}
	if err := s.record(context.WithoutCancel(ctx), e); err != nil {
		return err
	}
	return emit(e)
}

func (s *execution) rejectWave(ctx context.Context, calls []validatedCall, reused []bool, index int, outcome toolapi.Result, cause error) error {
	v := calls[index]
	outcome.Failed = true
	if outcome.Failure == nil {
		outcome.Failure = toolapi.FailureOf(cause)
	}
	var result error
	for i, call := range calls {
		if reused[i] {
			continue
		}
		if i == index {
			result = errors.Join(result, s.recordRejected(ctx, v.call, outcome, cause, s.emit), s.acceptResult(context.WithoutCancel(ctx), v, outcome, cause))
		} else {
			result = errors.Join(result, s.skipCalls(ctx, []validatedCall{call}, "Blocked by "+v.call.ID+" ("+v.call.Name+"); this call did not execute."))
		}
	}
	return result
}
