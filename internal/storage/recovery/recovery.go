// Purpose: Mark interrupted tasks and complete orphaned Tool Call records without replaying effects.
// Pattern: Recovery use case over a consumer-owned persistence contract.
// Usage: Called for the selected session before starting terminal workers.
// Related: internal/storage/recovery/state.go, internal/agent/record/checkpoint.go.
package recovery

import (
	"context"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func Restore(ctx context.Context, s Store, session string) error {
	if err := recoverRun(ctx, s, session); err != nil {
		return err
	}
	calls, err := s.PendingTools(ctx, session)
	if err != nil {
		return err
	}
	for _, call := range calls {
		message := modelapi.Message{Role: "tool", ToolCallID: call.ID, ToolName: call.Name, Content: `{"error":"operation interrupted; effects were not replayed; inspect current state before retrying"}`}
		event := runrecord.Event{Kind: "tool_interrupted", CallID: call.ID, Text: "Interrupted " + call.Name, Call: &call}
		if err = s.Commit(ctx, session, []runrecord.Entry{{Message: &message}, {Event: &event}}); err != nil {
			return err
		}
	}
	return nil
}
