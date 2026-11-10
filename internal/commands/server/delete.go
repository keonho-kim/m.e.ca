// Purpose: Block new writes and join owned readers before deleting a conversation.
// Pattern: Explicit destructive session transition.
// Usage: Native deleteSession is distinct from CloseSession resource release.
// Related: internal/storage/sqlite/delete.go, internal/commands/server/lifecycle.go.
package server

import (
	"context"
	"errors"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

func (s *Service) DeleteSession(ctx context.Context, id string) error {
	if !sessionstate.ValidID(id) {
		return fault.New(diag.InvalidArgument, diag.ServerExplicitSession)
	}
	if s.options.Delete == nil {
		return fault.New(diag.FailedPrecondition, diag.ServerWorkspaceUnavailable)
	}
	change, err := s.beginChange(ctx)
	if err != nil {
		return err
	}
	defer change.finish()
	s.mu.Lock()
	c := s.ownedConversation(id)
	if s.sessionRequest(id) != nil {
		s.mu.Unlock()
		return fault.New(diag.Conflict, diag.UIDeleteActive)
	}
	if s.conversation == c && c != nil {
		s.detachConversation()
	}
	delete(s.jobs, id)
	if s.last != nil && s.last.Session == id {
		s.last = nil
	}
	s.mu.Unlock()
	if err := s.closeReadersAndConversation(c); err != nil {
		return err
	}
	err = s.options.Delete(change.ctx, id)
	if err == nil {
		s.mu.Lock()
		s.publish(Event{Session: id, Record: eventKind("session_deleted")})
		s.mu.Unlock()
	}
	return errors.Join(err, change.ctx.Err())
}
