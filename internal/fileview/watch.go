// Purpose: Apply subscriber-specific path policy to a shared active-directory observation.
// Pattern: Cancellable subscription with joined resource release.
// Usage: Native file streams receive coalesced updates without duplicate kernel watchers.
// Related: internal/fileview/watch_pool.go, internal/commands/server/file_watch.go.
package fileview

import (
	"context"
	"errors"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/policy"
	"os"
	"path/filepath"
)

type FileChange struct {
	Refresh, Rescan bool
	Names           []string
}

func (e *Reader) Watch(ctx context.Context, settings policy.Settings, path string, notify func(FileChange) error) error {
	target, err := e.resolve(settings.Scope, path)
	if err != nil {
		return err
	}
	entry, updates, release, err := e.watches.borrow(target.Absolute)
	if err != nil {
		return err
	}
	defer release()
	if err := notify(FileChange{Refresh: true, Rescan: true}); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-entry.done:
			if entry.err != nil {
				return entry.err
			}
			return ctx.Err()
		case change := <-updates:
			if change.Refresh {
				current, err := e.resolve(settings.Scope, path)
				if err != nil {
					return err
				}
				if current.Absolute != target.Absolute {
					return fault.New(diag.StaleTarget, diag.LocalTargetChangedAfterPreparation)
				}
				filtered := FileChange{Refresh: true, Rescan: change.Rescan}
				for _, name := range change.Names {
					absolute := filepath.Join(target.Absolute, name)
					if _, err := e.resolve(settings.Scope, absolute); (err == nil || errors.Is(err, os.ErrNotExist)) && e.redact(absolute) == absolute {
						filtered.Names = append(filtered.Names, name)
					}
				}
				change = filtered
			}
			if err := notify(change); err != nil {
				return err
			}
		}
	}
}
