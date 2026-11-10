// Purpose: Recheck current workspace access before each file notification.
// Pattern: Consumer-owned observation contract.
// Usage: Native file streams own WatchFiles until their context is canceled.
// Related: internal/fileview/watch.go, internal/serverapi/files_watch.go.
package server

import (
	"context"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/policy"
)

type fileWatcher interface {
	Watch(context.Context, policy.Settings, string, func(bool) error) error
}

func (s *Service) WatchFiles(ctx context.Context, path string, notify func(bool) error) error {
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return err
	}
	defer done()
	files, settings, err := s.fileSettings()
	if err != nil {
		return err
	}
	watcher, ok := files.(fileWatcher)
	if !ok {
		return fault.New(diag.FailedPrecondition, diag.ServerWorkspaceUnavailable)
	}
	return watcher.Watch(ctx, settings, path, func(refresh bool) error {
		_, current, err := s.fileSettings()
		if err != nil {
			return err
		}
		if _, err := files.Reference(ctx, current, path); err != nil {
			return err
		}
		return notify(refresh)
	})
}
