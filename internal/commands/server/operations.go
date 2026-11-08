// Purpose: Reserve idle state changes and join cancellable I/O without holding the state mutex.
// Pattern: Explicit operation ownership and state reservation.
// Usage: Session, model, admission and read use cases hold a reservation or reader until cleanup finishes.
// Related: internal/commands/server/service.go, internal/commands/server/lifecycle.go.
package server

import (
	"context"

	"github.com/keonho-kim/meca/internal/config"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type sessionChange struct {
	ctx          context.Context
	conversation *Conversation
	config       config.Config
	finish       func()
}

// trackLocked requires s.mu; Close cannot begin waiting before registration.
func (s *Service) trackLocked(ctx context.Context) (context.Context, func()) {
	s.workers.Add(1)
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(s.ctx, cancel)
	return ctx, func() { stop(); cancel(); s.workers.Done() }
}

func (s *Service) beginOperation(ctx context.Context) (context.Context, func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if s.closed || s.ctx.Err() != nil {
		return nil, nil, faults.New(diagcatalog.Unavailable, diagcatalog.ServerServerIsShuttingDown)
	}
	ctx, done := s.trackLocked(ctx)
	return ctx, done, nil
}

func (s *Service) beginChange(ctx context.Context) (sessionChange, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return sessionChange{}, err
	}
	if err := s.requireIdle(); err != nil {
		return sessionChange{}, err
	}
	s.changing = true
	ctx, done := s.trackLocked(ctx)
	return sessionChange{ctx: ctx, conversation: s.conversation, config: s.options.Config, finish: func() {
		s.mu.Lock()
		s.changing, s.admitting = false, ""
		s.submitting = ""
		s.feed.wake()
		s.mu.Unlock()
		done()
	}}, nil
}

func (s *Service) beginRead(ctx context.Context) (*Conversation, context.Context, func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, nil, nil, err
	}
	if err := s.requireConversation(); err != nil {
		return nil, nil, nil, err
	}
	if s.changing {
		return nil, nil, nil, faults.New(diagcatalog.Conflict, diagcatalog.ServerSessionConfigurationIsChanging)
	}
	return s.borrowReadLocked(ctx, s.conversation)
}

// borrowReadLocked registers the reader before a close can start waiting.
func (s *Service) borrowReadLocked(ctx context.Context, c *Conversation) (*Conversation, context.Context, func(), error) {
	if c.readContext == nil {
		c.readContext, c.stopReaders = context.WithCancel(s.ctx)
	}
	c.readers.Add(1)
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(c.readContext, cancel)
	return c, ctx, func() { stop(); cancel(); c.readers.Done() }, nil
}
