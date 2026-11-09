// Purpose: Pair skipped native calls with recovery guidance appropriate to their actual cause.
// Pattern: Ordered non-execution feedback.
// Usage: Batch admission, rejected preparation and execution waves preserve completed siblings.
// Related: internal/agent/run/batch.go, internal/agent/run/rejected.go.
package run

import (
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"

	"context"
	"encoding/json"
	"errors"
	"time"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

type skipReason uint8

const (
	skipAfterFailure skipReason = iota
	skipForInstruction
	skipForClosure
	skipForContext
	skipForInterruption
)

func skippedFailure(reason skipReason, detail string) *faults.Failure {
	if reason == skipForInstruction {
		failure := faults.NewInstructionChanged()
		failure.Detail = detail
		return failure
	}
	failure := &faults.Failure{Code: diagcatalog.BatchNotExecuted, Detail: detail,
		Next: diagcatalog.Text(diagcatalog.RunPreserveCompletedSiblingResultsInspectTheBlocking)}
	switch reason {
	case skipForClosure:
		failure.Code = diagcatalog.SessionClosed
		failure.Next = diagcatalog.Text(diagcatalog.ClosedSessionRecovery)
	case skipForContext:
		failure.Next = diagcatalog.Text(diagcatalog.ContextFailureRecovery)
	case skipForInterruption:
		failure.Next = diagcatalog.Text(diagcatalog.InterruptedBatchRecovery)
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
