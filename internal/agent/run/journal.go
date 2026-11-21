// Purpose: Persist execution messages and checkpoint snapshots with provenance and the active uncompacted context.
// Pattern: Simple Module (journal I/O boundary).
// Usage: Called by execution phases and the final cancellation-safe flush.
// Related: internal/agent/record/events.go, internal/agent/run/run.go, internal/agent/run/statistics.go, internal/storage/sqlite/events.go.
package run

import (
	"context"
	"errors"
	"slices"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func (s *execution) save(ctx context.Context, m modelapi.Message, events ...runrecord.Event) error {
	records := make([]runrecord.Entry, len(events)+1)
	messageIndex, eventIndex := 0, 1
	if m.Incomplete {
		// An interrupted reply's Thinking precedes its text in the transcript.
		messageIndex, eventIndex = len(events), 0
	}
	records[messageIndex] = runrecord.Entry{Message: &m}
	for i := range events {
		records[eventIndex+i] = runrecord.Entry{Event: &events[i]}
	}
	return s.saveEntries(ctx, records)
}

func (s *execution) saveDecision(ctx context.Context, reply modelapi.Response, observedThrough int) error {
	if err := s.save(ctx, modelapi.Message{Role: "assistant", Content: reply.Text, Reasoning: reply.Reasoning, ReasoningState: reply.ReasoningState, Calls: reply.Calls}); err != nil {
		return err
	}
	s.acknowledgeObservations(observedThrough)
	return nil
}

func (s *execution) saveEntries(ctx context.Context, records []runrecord.Entry) error {
	for _, record := range records {
		if m := record.Message; m != nil {
			s.setMessageProvenance(m)
		}
		if event := record.Event; event != nil {
			*event = s.present(*event)
			event.RunID = s.checkpoint.ID
			if event.OccurredAt == "" {
				event.OccurredAt = time.Now().UTC().Format(time.RFC3339Nano)
			}
		}
	}
	if err := s.runner.Journal.Commit(ctx, s.runner.Session, records); err != nil {
		return errors.Join(errRuntimeRecord, err)
	}
	for _, record := range records {
		if record.Message != nil {
			s.messages = append(s.messages, *record.Message)
		}
	}
	return nil
}

func (s *execution) setMessageProvenance(m *modelapi.Message) {
	if m.Role == "assistant" {
		m.Presentation = s.presentation
	} else if m.Role == "user" && m.Presentation == "" {
		m.Presentation = s.order.Next()
	}
	m.Content = s.runner.Redact(m.Content)
	m.Reasoning = s.runner.Redact(m.Reasoning)
	m.Images = slices.Clone(m.Images)
	for i := range m.Images {
		m.Images[i].Path = s.runner.Redact(m.Images[i].Path)
	}
	m.ModelID, m.ProviderID = s.runner.Model, s.runner.Provider
	m.Thinking = s.runner.Thinking
	if m.Thinking != nil && *m.Thinking {
		m.ReasoningEffort = s.runner.ReasoningEffort
	}
}

func (s *execution) checkpointEvent() (runrecord.Event, error) {
	if err := s.checkpoint.Check(); err != nil {
		return runrecord.Event{}, err
	}
	copy := s.checkpoint
	event := runrecord.Event{Kind: "run_checkpoint", RunID: copy.ID, Checkpoint: &copy}
	if copy.Phase == "finished" {
		statistics := s.statistics.snapshot(time.Now())
		event.Statistics = &statistics
	}
	return event, nil
}

func (s *execution) persist(ctx context.Context, events ...runrecord.Event) error {
	checkpoint, err := s.checkpointEvent()
	if err != nil {
		return err
	}
	events = append([]runrecord.Event{checkpoint}, events...)
	records := make([]runrecord.Entry, len(events))
	for i := range events {
		events[i] = s.present(events[i])
		events[i].RunID = s.checkpoint.ID
		if events[i].OccurredAt == "" {
			events[i].OccurredAt = time.Now().UTC().Format(time.RFC3339Nano)
		}
		records[i].Event = &events[i]
	}
	if err := s.runner.Journal.Commit(ctx, s.runner.Session, records); err != nil {
		return errors.Join(errRuntimeRecord, err)
	}
	if checkpoint.Statistics != nil {
		return s.emit(checkpoint)
	}
	return nil
}
