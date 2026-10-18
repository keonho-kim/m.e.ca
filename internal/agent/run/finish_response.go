// Purpose: Commit main-agent answers after checking native-call syntax and current file facts.
// Pattern: Terminal state transition.
// Usage: The run loop calls finishResponse for replies without native tool calls.
// Related: internal/agent/run/run.go, internal/agent/run/finish.go.
package run

import (
	"context"

	"github.com/keonho-kim/v-code/internal/agent/response"
)

func (s *execution) finishResponse(ctx context.Context, text string) (bool, error) {
	if (len(s.checkpoint.Work.Receipts) > 0 || s.result.HadToolFailure) && response.ToolMarkup(text) {
		if err := s.record(ctx, Event{Kind: "response_rejected", Internal: true, Text: s.runner.Redact(text)}); err != nil {
			return false, err
		}
		if s.textCorrection {
			_, err := s.report(ctx, "The model repeated textual tool syntax; no such tool was executed.", true)
			return true, err
		}
		s.textCorrection = true
		s.checkpoint.Observation = "Your text contained a tool invocation that was NOT executed. Use the supplied native function schemas for remaining work, or return a plain user-facing final answer."
		return false, s.persist(ctx)
	}
	if err := s.refreshFileFacts(ctx); err != nil {
		return false, err
	}
	return s.finish(ctx, text)
}
