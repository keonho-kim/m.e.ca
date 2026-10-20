// Purpose: Persist execution messages and checkpoint snapshots with provenance and the active uncompacted context.
// Pattern: Simple Module (journal I/O boundary).
// Usage: Called by execution phases and the final cancellation-safe flush.
// Related: internal/agent/run/events.go, internal/agent/run/run.go, internal/agent/run/statistics.go, internal/storage/sqlite/events.go.
package run

import (
	"context"
	"errors"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"time"
)

func (s *execution) save(ctx context.Context, m modelapi.Message) error {
	m.Content = s.runner.Redact(m.Content)
	m.Reasoning = s.runner.Redact(m.Reasoning)
	m.ModelID, m.ProviderID = s.runner.Model, s.runner.Provider
	m.Thinking = s.runner.Thinking
	if m.Thinking != nil && *m.Thinking {
		m.ReasoningEffort = s.runner.ReasoningEffort
	}
	if err := s.runner.Journal.AppendMessage(ctx, s.runner.Session, m); err != nil {
		return errors.Join(errRuntimeRecord, err)
	}
	s.messages = append(s.messages, m)
	return nil
}

func (s *execution) persist(ctx context.Context) error {
	if err := s.checkpoint.Check(); err != nil {
		return err
	}
	copy := s.checkpoint
	event := Event{Kind: "run_checkpoint", RunID: copy.ID, Checkpoint: &copy}
	if copy.Phase == "finished" {
		statistics := s.statistics.snapshot(time.Now())
		event.Statistics = &statistics
	}
	if err := s.record(ctx, event); err != nil {
		return err
	}
	if event.Statistics != nil {
		return s.emit(event)
	}
	return nil
}
