// Purpose: Own detached native sessions while their tasks or detail readers remain active.
// Pattern: Session resource registry with explicit reader leases.
// Usage: Selection changes detach running sessions; completion retires inactive resources.
// Related: internal/commands/server/lifecycle.go, internal/commands/server/snapshot.go.
package server

import (
	"context"
	"errors"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

// idleJobs requires s.mu and detaches entries before cleanup can block.
func (s *Service) idleJobs(keep *Conversation) []*Conversation {
	var values []*Conversation
	for id, c := range s.jobs {
		if c != keep && c != s.conversation && s.sessionRequest(id) == nil {
			delete(s.jobs, id)
			values = append(values, c)
		}
	}
	if len(values) > len(s.jobs) {
		remaining := make(map[string]*Conversation, len(s.jobs))
		for id, c := range s.jobs {
			remaining[id] = c
		}
		s.jobs = remaining
	}
	return values
}

func (s *Service) beginSessionRead(ctx context.Context, id string) (*Conversation, context.Context, func(), error) {
	if !sessionstate.ValidID(id) {
		return nil, nil, nil, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerExplicitSession)
	}
	s.mu.Lock()
	if s.closed || ctx.Err() != nil {
		s.mu.Unlock()
		return nil, nil, nil, context.Canceled
	}
	c := s.ownedConversation(id)
	if c != nil {
		c, ctx, done, err := s.borrowReadLocked(ctx, c)
		s.mu.Unlock()
		return c, ctx, done, err
	}
	ctx, done := s.trackLocked(ctx)
	read := s.options.Read
	s.mu.Unlock()
	if read == nil {
		done()
		return nil, nil, nil, faults.New(diagcatalog.NotFound, diagcatalog.ServerConversationIsNotCurrentlyOwned)
	}
	c, err := read(ctx, id)
	if err != nil {
		done()
		return nil, nil, nil, err
	}
	return c, ctx, func() {
		closeErr := closeConversation(c)
		s.mu.Lock()
		s.closeErr = errors.Join(s.closeErr, closeErr)
		s.mu.Unlock()
		done()
	}, nil
}
