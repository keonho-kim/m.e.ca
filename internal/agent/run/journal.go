// Purpose: Persist task messages and checkpoint snapshots with provenance and bounded in-memory history.
// Pattern: Simple Module (journal I/O boundary).
// Usage: Called by task phases and the final cancellation-safe flush.
// Related: internal/agent/run/types.go, internal/agent/run/run.go, internal/storage/sqlite/events.go.
package run

import (
	"context"
	"github.com/keonho-kim/v-code/internal/agent"
	"github.com/keonho-kim/v-code/internal/agent/compact"
)

func (s *task) save(ctx context.Context, m agent.Message) error {
	m.Content = s.runner.Redact(m.Content)
	m.ModelID, m.ProviderID = s.runner.Model, s.runner.Provider
	m.Thinking = s.runner.Thinking
	m.Planning = s.planning
	if err := s.runner.Journal.AppendMessage(ctx, s.runner.Session, m); err != nil {
		return err
	}
	if m.Role == "user" || m.Role == "tool" {
		if err := s.observe(ctx, compact.Estimate(m.Content, m.Role)); err != nil {
			return err
		}
	}
	s.result.Messages = append(s.result.Messages, m)
	if len(s.result.Messages) > 1000 {
		s.result.Messages = append([]agent.Message(nil), s.result.Messages[len(s.result.Messages)-1000:]...)
	}
	return nil
}

func (s *task) persist(ctx context.Context) error {
	if err := s.checkpoint.Check(); err != nil {
		return err
	}
	copy := s.checkpoint
	return s.runner.Journal.RecordEvent(ctx, s.runner.Session, Event{Kind: "task_checkpoint", Checkpoint: &copy})
}
