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
	"os"
	"path/filepath"
	"sort"
	"time"
)

type FileChange struct {
	Refresh, Rescan bool
	Names           []string
}

func (e *Reader) Watch(ctx context.Context, settings policy.Settings, path string, notify func(FileChange) error) (resultErr error) {
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
	if err := notify(FileChange{Refresh: true, Rescan: true}); err != nil {
		return err
	}
	dirty := false
	names, rescan := make(map[string]struct{}), false
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-watcher.Events:
			if !ok {
				return ctx.Err()
			}
			relative, pathErr := filepath.Rel(t.Absolute, event.Name)
			if pathErr != nil || relative == "." || !filepath.IsLocal(relative) || filepath.Dir(relative) != "." || len(names) >= 128 {
				rescan = true
			} else if _, err := e.resolve(settings.Scope, event.Name); (err == nil || errors.Is(err, os.ErrNotExist)) && e.redact(event.Name) == event.Name {
				names[relative] = struct{}{}
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
				rescan = true
				if !dirty {
					dirty = true
					timer.Reset(150 * time.Millisecond)
				}
			} else {
				return err
			}
		case <-timer.C:
			change := FileChange{Refresh: dirty, Rescan: rescan}
			for name := range names {
				change.Names = append(change.Names, name)
			}
			sort.Strings(change.Names)
			if err := notify(change); err != nil {
				return err
			}
			if dirty {
				timer.Reset(5 * time.Second)
			} else {
				timer.Reset(time.Second)
			}
			dirty = false
			clear(names)
			rescan = false
		}
	}
}
