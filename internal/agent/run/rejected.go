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
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"time"
)

func (s *execution) recordRejected(ctx context.Context, v validatedCall, outcome toolapi.Result, cause error, emit func(runrecord.Event) error) error {
	call := v.call
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
	e := runrecord.Event{Kind: "tool_failed", CallID: call.ID, Call: &call, Text: s.runner.Redact(failure.Error()), Output: body}
	outcome.Failed, outcome.Failure = true, failure
	recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := s.acceptResult(recordCtx, v, outcome, cause, e); err != nil {
		return errors.Join(errRuntimeRecord, err)
	}
	return emit(e)
}

func (s *execution) rejectWave(ctx context.Context, calls []validatedCall, index int, outcome toolapi.Result, cause error) error {
	v := calls[index]
	outcome.Failed = true
	if outcome.Failure == nil {
		outcome.Failure = toolapi.FailureOf(cause)
	}
	var result error
	for i, call := range calls {
		if i == index {
			result = errors.Join(result, s.recordRejected(ctx, v, outcome, cause, s.emit))
		} else {
			result = errors.Join(result, s.skipCalls(ctx, []validatedCall{call}, "Blocked by "+v.call.ID+" ("+v.call.Name+"); this call did not execute."))
		}
	}
	return result
}
