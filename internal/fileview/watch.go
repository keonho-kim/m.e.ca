// Purpose: Watch one authorized directory without retaining a recursive project tree.
// Pattern: Caller-owned cancellable observation.
// Usage: Native file streams receive coalesced refresh requests and idle notifications.
// Related: internal/fileview/reader.go, internal/commands/server/file_watch.go.
package fileview

import (
	"context"
	"errors"
	"github.com/fsnotify/fsnotify"
	"github.com/keonho-kim/meca/internal/policy"
	"time"
)

func (e *Reader) Watch(ctx context.Context, settings policy.Settings, path string, notify func(bool) error) (resultErr error) {
	t, err := e.resolve(settings.Scope, path)
	if err != nil {
		return err
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, watcher.Close()) }()
	if err := watcher.Add(t.Absolute); err != nil {
		return err
	}
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	if err := notify(true); err != nil {
		return err
	}
	dirty := false
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case _, ok := <-watcher.Events:
			if !ok {
				return ctx.Err()
			}
			if !dirty {
				dirty = true
				timer.Reset(150 * time.Millisecond)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return ctx.Err()
			}
			if errors.Is(err, fsnotify.ErrEventOverflow) {
				if !dirty {
					dirty = true
					timer.Reset(150 * time.Millisecond)
				}
			} else {
				return err
			}
		case <-timer.C:
			if err := notify(dirty); err != nil {
				return err
			}
			if dirty {
				timer.Reset(5 * time.Second)
			} else {
				timer.Reset(time.Second)
			}
			dirty = false
		}
	}
}
