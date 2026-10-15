// Purpose: Persist execution messages and checkpoint snapshots with provenance and the active uncompacted context.
// Pattern: Simple Module (journal I/O boundary).
// Usage: Called by execution phases and the final cancellation-safe flush.
// Related: internal/agent/run/events.go, internal/agent/run/run.go, internal/storage/sqlite/events.go.
package run

import (
	"context"
	"github.com/keonho-kim/v-code/internal/agent/compact"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
)

func (s *execution) save(ctx context.Context, m modelapi.Message) error {
	m.Content = s.runner.Redact(m.Content)
	m.Reasoning = s.runner.Redact(m.Reasoning)
	m.ModelID, m.ProviderID = s.runner.Model, s.runner.Provider
	m.Thinking = s.runner.Thinking
	if m.Thinking != nil && *m.Thinking {
		m.ReasoningEffort = s.runner.ReasoningEffort
	}
	m.Planning = s.planning
	if err := s.runner.Journal.AppendMessage(ctx, s.runner.Session, m); err != nil {
		return err
	}
	if m.Role == "user" || m.Role == "tool" {
		if err := s.observe(ctx, compact.Estimate(m.Content, m.Role)); err != nil {
			return err
		}
	}
	s.messages = append(s.messages, m)
	return nil
}

func (s *execution) persist(ctx context.Context) error {
	if err := s.checkpoint.Check(); err != nil {
		return err
	}
	copy := s.checkpoint
	return s.record(ctx, Event{Kind: "run_checkpoint", Checkpoint: &copy})
}
