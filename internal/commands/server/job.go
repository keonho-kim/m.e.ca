// Purpose: Own one compatible execution conversation independently of native selection.
// Pattern: Bounded application resource slot with conversation-local reader cancellation.
// Usage: Submit replaces the idle slot; output reads borrow either owned conversation.
// Related: internal/commands/server/submission_session.go, internal/commands/server/lifecycle.go.
package server

import (
	"context"
	"errors"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s *Service) replaceJob(change sessionChange, next *Conversation) error {
	s.mu.Lock()
	old := s.job
	s.job = nil
	s.mu.Unlock()
	if old != nil && old != next {
		if err := s.closeReadersAndConversation(old); err != nil {
			return errors.Join(err, closeConversation(next))
		}
	}
	s.mu.Lock()
	if s.closed || change.ctx.Err() != nil {
		s.mu.Unlock()
		return errors.Join(context.Canceled, closeConversation(next))
	}
	s.job = next
	s.publish(Event{Session: next.Runtime.Session.ID, Record: eventKind("execution_session_opened")})
	s.mu.Unlock()
	return nil
}

func (s *Service) beginResultRead(ctx context.Context, id string) (*Conversation, context.Context, func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || ctx.Err() != nil {
		return nil, nil, nil, context.Canceled
	}
	c := s.conversation
	if c == nil || c.Runtime.Session.ID != id {
		c = s.job
	}
	if c == nil || c.Runtime.Session.ID != id {
		return nil, nil, nil, faults.New(diagcatalog.NotFound, diagcatalog.ServerConversationIsNotCurrentlyOwned)
	}
	return s.borrowReadLocked(ctx, c)
}
