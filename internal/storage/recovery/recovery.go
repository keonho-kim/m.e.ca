// Purpose: Mark interrupted tasks and complete orphaned Tool Call records without replaying effects.
// Pattern: Recovery use case over a consumer-owned persistence contract.
// Usage: Called for the selected session before starting terminal workers.
// Related: internal/storage/recovery/state.go, internal/agent/record/checkpoint.go.
package recovery

import (
	"context"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
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
			message := modelapi.Message{Role: "tool", ToolCallID: call.ID, ToolName: call.Name, Content: `{"error":"operation interrupted; effects were not replayed; inspect current state before retrying"}`}
			event := runrecord.Event{Kind: "tool_interrupted", CallID: call.ID, Text: "Interrupted " + call.Name, Call: &call}
			if err = s.Commit(ctx, session, []runrecord.Entry{{Message: &message}, {Event: &event}}); err != nil {
				return err
			}
			delete(pending, call.ID)
		}
	}
	return nil
}
