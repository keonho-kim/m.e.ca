// Purpose: Persist execution messages and checkpoint snapshots with provenance and the active uncompacted context.
// Pattern: Simple Module (journal I/O boundary).
// Usage: Called by execution phases and the final cancellation-safe flush.
// Related: internal/agent/run/events.go, internal/agent/run/run.go, internal/agent/run/statistics.go, internal/storage/sqlite/events.go.
package run

import (
	"context"
	"errors"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"time"
)

func (s *execution) save(ctx context.Context, m modelapi.Message, events ...Event) error {
	m.Content = s.runner.Redact(m.Content)
	m.Reasoning = s.runner.Redact(m.Reasoning)
	m.ModelID, m.ProviderID = s.runner.Model, s.runner.Provider
	m.Thinking = s.runner.Thinking
	if m.Thinking != nil && *m.Thinking {
		m.ReasoningEffort = s.runner.ReasoningEffort
	}
	records := []JournalRecord{{Message: &m}}
	for i := range events {
		events[i].RunID = s.checkpoint.ID
		records = append(records, JournalRecord{Event: &events[i]})
	}
	if err := s.runner.Journal.Commit(ctx, s.runner.Session, records); err != nil {
		return errors.Join(errRuntimeRecord, err)
	}
	s.messages = append(s.messages, m)
	return nil
}

func (s *execution) checkpointEvent() (Event, error) {
	if err := s.checkpoint.Check(); err != nil {
		return Event{}, err
	}
	copy := s.checkpoint
	event := Event{Kind: "run_checkpoint", RunID: copy.ID, Checkpoint: &copy}
	if copy.Phase == "finished" {
		statistics := s.statistics.snapshot(time.Now())
		event.Statistics = &statistics
	}
	return event, nil
}

func (s *execution) persist(ctx context.Context, events ...Event) error {
	event, err := s.checkpointEvent()
	if err != nil {
		return err
	}
	events = append([]Event{event}, events...)
	records := make([]JournalRecord, len(events))
	for i := range events {
		events[i].RunID = s.checkpoint.ID
		records[i].Event = &events[i]
	}
	if err := s.runner.Journal.Commit(ctx, s.runner.Session, records); err != nil {
		return errors.Join(errRuntimeRecord, err)
	}
	if event.Statistics != nil {
		return s.emit(event)
	}
	return nil
}
