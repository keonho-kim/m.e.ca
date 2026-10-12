// Purpose: Mark interrupted tasks and complete orphaned Tool Call records without replaying effects.
// Pattern: Repository recovery boundary.
// Usage: Called for the selected session before starting terminal workers.
// Related: internal/storage/sqlite/messages.go, internal/storage/sqlite/plans.go.
package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/keonho-kim/v-code/internal/agent"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/agent/run"
)

func (s *Store) Recover(ctx context.Context, session string) error {
	if err := s.recoverTask(ctx, session); err != nil {
		return err
	}
	messages, err := s.Messages(ctx, session, 1000)
	if err != nil {
		return err
	}
	pending := map[string]agent.ToolCall{}
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
			if err = s.AppendMessage(ctx, session, agent.Message{Role: "tool", ToolCallID: call.ID, ToolName: call.Name, Content: `{"error":"operation interrupted; effects were not replayed; inspect current state before retrying"}`}); err != nil {
				return err
			}
			delete(pending, call.ID)
			if err = s.RecordEvent(ctx, session, run.Event{Kind: "tool_interrupted", CallID: call.ID, Text: "Interrupted " + call.Name}); err != nil {
				return err
			}
		}
	}
	p, err := s.LatestPlan(ctx, session)
	if err != nil || p == nil {
		return err
	}
	if p.Status == planning.PlanRunning {
		for i, step := range p.Steps {
			if step.Status == "running" {
				next, err := planning.AdvanceStep(*p, i, "interrupted", time.Now())
				if err != nil {
					return fmt.Errorf("recover Plan: %w", err)
				}
				return s.SavePlan(ctx, next)
			}
		}
	}
	return nil
}
