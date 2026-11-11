// Purpose: Read persisted token totals without restoring conversation messages.
// Pattern: Read-only session lease over the usage projection.
// Usage: Native sessionUsage refreshes the composer status line after completed requests.
// Related: internal/storage/sqlite/usage.go, internal/commands/server/job.go.
package server

import (
	"context"

	"github.com/keonho-kim/meca/internal/agent/model"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

type usageReader interface {
	TokenUsage(context.Context, string) (model.TokenUsage, error)
}

func (s *Service) SessionUsage(ctx context.Context, session string) (model.TokenUsage, error) {
	if !sessionstate.ValidID(session) {
		return model.TokenUsage{}, fault.New(diag.InvalidArgument, diag.ServerExplicitSession)
	}
	if s.options.Usage != nil {
		ctx, done, err := s.beginOperation(ctx)
		if err != nil {
			return model.TokenUsage{}, err
		}
		defer done()
		return s.options.Usage(ctx, session)
	}
	c, ctx, done, err := s.beginSessionRead(ctx, session)
	if err != nil {
		return model.TokenUsage{}, err
	}
	defer done()
	reader, ok := c.History.(usageReader)
	if !ok {
		return model.TokenUsage{}, fault.New(diag.FailedPrecondition, diag.ServerWorkspaceUnavailable)
	}
	return reader.TokenUsage(ctx, session)
}
