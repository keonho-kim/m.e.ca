// Purpose: Admit ordered collaboration observations with an atomic durable receive cursor.
// Pattern: Optional input boundary for cooperative sessions.
// Usage: Team supplies PollContext; ordinary runs incur no repository calls here.
// Related: internal/agent/run/loop.go, internal/storage/sqlite/commit.go.
package run

import (
	"context"
	"fmt"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

type ContextBatch struct {
	Messages  []modelapi.Message
	Cursor    int64
	Objective string
}

type contextJournal interface {
	CommitContext(context.Context, string, []runrecord.Entry, int64) error
}

func (s *execution) admitContext(ctx context.Context) (bool, error) {
	if s.runner.PollContext == nil {
		return false, nil
	}
	batch, err := s.runner.PollContext(ctx)
	if err != nil || batch.Cursor == 0 {
		return false, err
	}
	journal, ok := s.runner.Journal.(contextJournal)
	if !ok {
		return false, fmt.Errorf("cooperative context requires an atomic receive cursor")
	}
	entries := make([]runrecord.Entry, len(batch.Messages))
	for i := range batch.Messages {
		message := &batch.Messages[i]
		message.Content = s.runner.Redact(message.Content)
		message.ModelID, message.ProviderID = s.runner.Model, s.runner.Provider
		if err := message.Check(); err != nil {
			return false, err
		}
		entries[i].Message = message
	}
	if err := journal.CommitContext(ctx, s.runner.Session, entries, batch.Cursor); err != nil {
		return false, err
	}
	s.messages = append(s.messages, batch.Messages...)
	if batch.Objective != "" {
		s.checkpoint.Objective = s.runner.Redact(batch.Objective)
	}
	return len(batch.Messages) > 0, nil
}

func (s *execution) continueContext(ctx context.Context, reply modelapi.Response) (bool, error) {
	if s.runner.PollContext == nil {
		return false, nil
	}
	changed, err := s.admitContext(ctx)
	if err != nil || !changed {
		return changed, err
	}
	if reply.Text != "" {
		if err := s.save(ctx, modelapi.Message{Role: "assistant", Content: reply.Text, Reasoning: reply.Reasoning, ReasoningState: reply.ReasoningState}); err != nil {
			return false, err
		}
		if err := s.emit(runrecord.Event{Kind: "progress", Text: reply.Text}); err != nil {
			return false, err
		}
	}
	return true, nil
}
