// Purpose: Mark interrupted tasks and complete orphaned Tool Call records without replaying effects.
// Pattern: Recovery use case over a consumer-owned persistence contract.
// Usage: Called for the selected session before starting terminal workers.
// Related: internal/storage/recovery/state.go, internal/agent/run/checkpoint.go.
package recovery

import (
	"context"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/run"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

func Restore(ctx context.Context, s Store, session string) error {
	if err := recoverRun(ctx, s, session); err != nil {
		return err
	}
	messages, err := s.Messages(ctx, session, 1000)
	if err != nil {
		return err
	}
	pending := map[string]toolapi.Call{}
	for _, m := range messages {
		for _, call := range m.Calls {
			pending[call.ID] = call
		}
		if m.Role == "tool" {
			delete(pending, m.ToolCallID)
		}
	}
	for _, m := range messages {
		for _, call := range m.Calls {
			if _, ok := pending[call.ID]; !ok {
				continue
			}
			if err = s.AppendMessage(ctx, session, modelapi.Message{Role: "tool", ToolCallID: call.ID, ToolName: call.Name, Content: `{"error":"operation interrupted; effects were not replayed; inspect current state before retrying"}`}); err != nil {
				return err
			}
			delete(pending, call.ID)
			if err = s.RecordEvent(ctx, session, run.Event{Kind: "tool_interrupted", CallID: call.ID, Text: "Interrupted " + call.Name, Call: &call}); err != nil {
				return err
			}
		}
	}
	return nil
}
