// Purpose: Pair skipped native calls with recovery guidance appropriate to their actual cause.
// Pattern: Ordered non-execution feedback.
// Usage: Batch admission, rejected preparation and execution waves preserve completed siblings.
// Related: internal/agent/run/batch.go, internal/agent/run/rejected.go.
package run

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type skipReason uint8

const (
	skipAfterFailure skipReason = iota
	skipForInstruction
	skipForClosure
	skipForContext
	skipForInterruption
)

func skippedFailure(reason skipReason, detail string) *toolapi.Failure {
	if reason == skipForInstruction {
		failure := toolapi.NewInstructionChanged()
		failure.Detail = detail
		return failure
	}
	failure := &toolapi.Failure{Code: "batch_not_executed", Detail: detail,
		Next: "Preserve completed sibling results. Inspect the blocking outcome and the latest user request, then submit only still-needed work; do not replay successful changes."}
	switch reason {
	case skipForClosure:
		failure.Code = "session_closed"
		failure.Next = "The session has closed. Do not retry these calls. Its completed work and final report remain in the recorded conversation."
	case skipForContext:
		failure.Next = "Restore the failed context or journal access before making another decision. Inspect saved outcomes and the latest user instructions before considering any remaining work."
	case skipForInterruption:
		failure.Next = "Execution was interrupted or its budget ended. Inspect saved outcomes and any partial effects before resuming only work the user still requires."
	}
	return failure
}

func (s *execution) skipCalls(ctx context.Context, calls []validatedCall, reason skipReason, detail string) error {
	recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	var result error
	for _, v := range calls {
		failure := skippedFailure(reason, detail)
		body, err := json.Marshal(map[string]any{"executed": false, "call_id": v.call.ID, "tool": v.call.Name, "status": "skipped", "cause": detail})
		if err != nil {
			return err
		}
		outcome := toolapi.Result{Failed: true, Failure: failure, Content: body}
		event := runrecord.Event{Kind: "tool_failed", CallID: v.call.ID, Call: &v.call, Text: failure.Error(), Output: outcome.Content}
		if err := s.acceptResult(recordCtx, v, outcome, failure, event); err != nil {
			return errors.Join(result, errRuntimeRecord, err)
		}
		if err := s.emit(event); err != nil {
			result = errors.Join(result, err)
		}
	}
	return result
}
