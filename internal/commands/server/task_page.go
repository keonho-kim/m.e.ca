// Purpose: Reconcile durable task pages and storage usage with live engine ownership.
// Pattern: Application read use case.
// Usage: Native task list and capabilities expose durable history and conservative quota usage.
// Related: internal/storage/serverrequests/page.go, internal/commands/server/admission.go.
package server

import (
	"context"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type taskPages interface {
	Page(context.Context, string, string, int) ([]Task, string, error)
	Usage(context.Context) (int64, int64, error)
}

func (s *Service) Tasks(ctx context.Context, session, cursor string, limit int) ([]Task, string, error) {
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return nil, "", err
	}
	defer done()
	store, ok := s.options.Ledger.(taskPages)
	if !ok {
		return nil, "", faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerWorkspaceUnavailable)
	}
	values, next, err := store.Page(ctx, session, cursor, limit)
	if err != nil {
		return nil, "", err
	}
	for i := range values {
		values[i] = s.observedTask(values[i])
	}
	if s.options.Summaries != nil && len(values) != 0 {
		ids := make([]string, len(values))
		for i := range values {
			ids[i] = values[i].Session
		}
		titles, err := s.options.Summaries(ctx, ids)
		if err != nil {
			return nil, "", err
		}
		for i := range values {
			values[i].SessionSummary = titles[values[i].Session]
		}
	}
	return values, next, nil
}

func (s *Service) StorageUsage(ctx context.Context) (int64, int64, error) {
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer done()
	store, ok := s.options.Ledger.(taskPages)
	if !ok {
		return 0, 0, nil
	}
	return store.Usage(ctx)
}
