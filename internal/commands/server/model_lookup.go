// Purpose: Scope provider metadata reads to the selected credential reference.
// Pattern: Request-owned model lookup lease.
// Usage: DefaultModel, Discover and Diagnose share a cancellable configuration read.
// Related: internal/commands/server/discovery.go, internal/commands/server/diagnose.go, internal/commands/server/operations.go.
package server

import (
	"context"

	"github.com/keonho-kim/meca/internal/provider/connection"
)

type modelLookup struct {
	ctx           context.Context
	finish        func()
	previous      connection.Selection
	contextLength int
}

func (s *Service) beginModelLookup(ctx context.Context, session string) (modelLookup, error) {
	var c *Conversation
	var finish func()
	var err error
	if session == "" {
		ctx, finish, err = s.beginOperation(ctx)
	} else {
		c, ctx, finish, err = s.beginSessionRead(ctx, session)
	}
	if err != nil {
		return modelLookup{}, err
	}
	s.mu.Lock()
	cfg := s.currentConfig()
	var previous connection.Selection
	if c == nil {
		previous, err = configuredSelection(cfg.Model)
	} else {
		previous, err = c.modelSelection(cfg.Model)
	}
	s.mu.Unlock()
	if err != nil {
		finish()
		return modelLookup{}, err
	}
	return modelLookup{ctx: ctx, finish: finish, previous: previous, contextLength: cfg.ContextLength}, nil
}
