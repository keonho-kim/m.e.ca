// Purpose: Own a single filesystem subscription, registration and coalescing refresh timer.
// Pattern: Event-loop-owned lifecycle with joined workers.
// Usage: The terminal session submits reads and accepts immutable results.
// Related: internal/tui/session/directory.go, internal/tui/session/file_preview.go, internal/tui/browser/readqueue/slot.go.
package controller

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

type WatchRequest struct {
	Path     string
	Revision int
}

type WatchResult struct {
	WatchRequest
	Watcher *fsnotify.Watcher
	Err     error
}

type Watch struct {
	Watcher             *fsnotify.Watcher
	Path                string
	Events              <-chan fsnotify.Event
	Errors              <-chan error
	Timer               *time.Timer
	Ticks               <-chan time.Time
	Dirty, PreviewDirty bool
}

func (c *Controller) SyncWatch(parent context.Context, sink Sink, path string) error {
	if c.closed {
		return context.Canceled
	}
	if path != c.Watch.Path {
		if err := c.StopWatch(); err != nil {
			return err
		}
		c.Watch.Path = path
		request := WatchRequest{Path: path, Revision: c.WatchRevision}
		if c.WatchReads.Submit(request) {
			c.startWatch(parent, sink, request)
		}
	}
	if c.Watch.Dirty && c.Watch.Ticks == nil && !c.Directory.Running() {
		c.ScheduleRefresh()
	}
	return nil
}
func (c *Controller) StopWatch() error {
	c.WatchReads.Stop()
	c.WatchRevision++
	if c.Watch.Timer != nil {
		c.Watch.Timer.Stop()
	}
	var err error
	if c.Watch.Watcher != nil {
		err = c.Watch.Watcher.Close()
	}
	c.Watch = Watch{}
	return err
}
func (c *Controller) startWatch(parent context.Context, sink Sink, request WatchRequest) {
	owner := c.begin(parent, sink)
	if owner == nil {
		return
	}
	ctx, cancel := context.WithCancel(owner)
	c.WatchReads.Begin(request.Revision, cancel)
	c.workers.Go(func() {
		defer cancel()
		result := WatchResult{WatchRequest: request}
		result.Watcher, result.Err = fsnotify.NewWatcher()
		if result.Err == nil && ctx.Err() == nil {
			result.Err = result.Watcher.Add(request.Path)
		}
		if ctx.Err() != nil {
			result.Err = ctx.Err()
		}
		if result.Err != nil && result.Watcher != nil {
			result.Watcher.Close()
			result.Watcher = nil
		}
		if err := c.sink.DeliverFiles(owner, Result{Watch: &result}); err != nil && result.Watcher != nil {
			result.Watcher.Close()
		}
	})
}
func (c *Controller) AcceptWatch(parent context.Context, sink Sink, result *WatchResult, visible, previewOpen bool) error {
	if next := c.WatchReads.Complete(result.Revision); next != nil && visible && parent.Err() == nil {
		c.startWatch(parent, sink, *next)
	}
	if result.Revision != c.WatchRevision || !visible || parent.Err() != nil {
		if result.Watcher != nil {
			result.Watcher.Close()
		}
		return nil
	}
	if result.Err != nil {
		return result.Err
	}
	c.Watch.Watcher, c.Watch.Events, c.Watch.Errors = result.Watcher, result.Watcher.Events, result.Watcher.Errors
	c.Watch.Dirty, c.Watch.PreviewDirty = true, previewOpen
	return nil
}
func (c *Controller) WatchFailed(err error, previewOpen bool) error {
	if errors.Is(err, fsnotify.ErrEventOverflow) {
		c.Watch.Dirty, c.Watch.PreviewDirty = true, previewOpen
		c.ScheduleRefresh()
		return nil
	}
	path := c.Watch.Path
	closeErr := c.StopWatch()
	c.Watch.Path = path
	return errors.Join(err, closeErr)
}
func (c *Controller) Observe(event fsnotify.Event, previewPath string) {
	if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) == 0 {
		return
	}
	c.Watch.Dirty = true
	if previewPath != "" && filepath.Clean(event.Name) == filepath.Clean(previewPath) {
		c.Watch.PreviewDirty = true
	}
	c.ScheduleRefresh()
}
func (c *Controller) ScheduleRefresh() {
	const delay = 150 * time.Millisecond
	if c.Watch.Timer == nil {
		c.Watch.Timer = time.NewTimer(delay)
	} else {
		c.Watch.Timer.Reset(delay)
	}
	c.Watch.Ticks = c.Watch.Timer.C
}
func (c *Controller) Refresh(path string) (bool, bool) {
	c.Watch.Ticks = nil
	if c.Directory.Running() || !c.Watch.Dirty || path != c.Watch.Path {
		return false, false
	}
	c.Watch.Dirty = false
	preview := c.Watch.PreviewDirty
	c.Watch.PreviewDirty = false
	return true, preview
}
