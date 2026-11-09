// Purpose: Admit ordered collaboration observations with an atomic durable receive cursor.
// Pattern: Optional input boundary for cooperative sessions.
// Usage: Team supplies PollContext; ordinary runs incur no repository calls here.
// Related: internal/agent/run/loop.go, internal/storage/sqlite/commit.go.
package run

import (
	"context"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type ContextBatch struct {
	Messages     []modelapi.Message
	Cursor       int64
	UserSequence int64
	Objective    string
	Reconsider   bool
}

type contextJournal interface {
	CommitContext(context.Context, string, []runrecord.Entry, int64) error
}

// admitContext preserves all received content and returns only whether another decision is needed.
func (s *execution) admitContext(ctx context.Context) (bool, error) {
	if s.runner.PollContext == nil {
		return false, nil
	}
	batch, err := s.runner.PollContext(ctx)
	if err != nil {
		return false, err
	}
	if batch.UserSequence < 0 || batch.Objective != "" && batch.UserSequence == 0 {
		return false, faults.Errorf(diagcatalog.RunUserContextRequiresItsCommittedPublicSequence)
	}
	if batch.Cursor == 0 {
		s.observeUserSequence(batch.UserSequence)
		return batch.Reconsider, nil
	}
	journal, ok := s.runner.Journal.(contextJournal)
	if !ok {
		return false, faults.Errorf(diagcatalog.RunCooperativeContextRequiresAnAtomicReceiveCursor)
	}
	entries := make([]runrecord.Entry, len(batch.Messages))
	for i := range batch.Messages {
		message := &batch.Messages[i]
		clean := s.runner.Redact(message.Content)
		if clean != message.Content {
			// A sanitized local observation must not reference different original bytes.
			message.ID += "#local"
			if message.Source != nil {
				local := *message.Source
				local.Sequence = 0
				local.Session = ""
				message.Source = &local
			}
		}
		message.Content = clean
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
	s.observeUserSequence(batch.UserSequence)
	if batch.Objective != "" {
		s.checkpoint.Objective = s.runner.Redact(batch.Objective)
		if err := s.readMentions(ctx, batch.Objective); err != nil {
			return false, err
		}
	}
	return batch.Reconsider || batch.Objective != "", nil
}

func (s *execution) observeUserSequence(sequence int64) {
	if sequence > s.userSequence {
		s.userSequence = sequence
		s.instructionRevision++
	}
}

func (s *execution) continueContext(ctx context.Context, reply modelapi.Response) (bool, error) {
	if s.runner.PollContext == nil {
		return false, nil
	}
	reconsider, err := s.admitContext(ctx)
	if err != nil || !reconsider {
		return reconsider, err
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
