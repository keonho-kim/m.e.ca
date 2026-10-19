// Purpose: Return rejected native candidates as paired tool results without performing side effects.
// Pattern: Native tool dialogue recovery.
// Usage: Decision and action handlers reject invalid schemas, multi-call batches and deferred candidates here.
// Related: internal/agent/run/action.go, internal/agent/run/journal.go, internal/agent/context/conversation.go.
package run

import (
	"context"
	"encoding/json"
	"fmt"

	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

func (s *execution) returnToolFeedback(ctx context.Context, reply modelapi.Response, detail string) error {
	calls := reply.Calls
	if len(calls) == 0 || len(calls) > toolapi.MaxCalls {
		return fmt.Errorf("invalid native call count")
	}
	for _, call := range calls {
		if err := call.Check(); err != nil {
			return err
		}
	}
	if err := s.save(ctx, modelapi.Message{Role: "assistant", Content: reply.Text, Reasoning: reply.Reasoning, Calls: calls}); err != nil {
		return err
	}
	for _, call := range calls {
		body, err := json.Marshal(map[string]any{"executed": false, "feedback": s.runner.Redact(detail)})
		if err != nil {
			return err
		}
		if err := s.save(ctx, modelapi.Message{Role: "tool", Content: string(body), ToolCallID: call.ID, ToolName: call.Name}); err != nil {
			return err
		}
		event := Event{Kind: "tool_failed", Text: "Not executed: " + s.runner.Redact(detail), CallID: call.ID, Call: &call, Output: body}
		if err := s.record(ctx, event); err != nil {
			return err
		}
		if err := s.emit(event); err != nil {
			return err
		}
	}
	s.result.HadToolFailure = true
	s.checkpoint.Observation = s.runner.Redact(detail)
	return s.persist(ctx)
}
