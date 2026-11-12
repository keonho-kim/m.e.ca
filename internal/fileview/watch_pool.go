// Purpose: Share one kernel watcher and coalescing clock among active subscribers to a directory.
// Pattern: Reference-counted observation resource with joined shutdown.
// Usage: Reader.Watch borrows a directory stream and owns its subscription release.
// Related: internal/fileview/watch.go, internal/fileview/reader.go.
package fileview

import (
	"context"
	"errors"
	"github.com/fsnotify/fsnotify"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type directoryWatches struct {
	mu      sync.Mutex
	entries map[string]*directoryWatch
}
type directoryWatch struct {
	cancel      context.CancelFunc
	done        chan struct{}
	subscribers map[chan FileChange]struct{}
	err         error
}

func (p *directoryWatches) borrow(path string) (*directoryWatch, chan FileChange, func(), error) {
	p.mu.Lock()
	if p.entries == nil {
		p.entries = make(map[string]*directoryWatch)
	}
	entry := p.entries[path]
	if entry != nil {
		select {
		case <-entry.done:
			entry = nil
		default:
		}
	}
	if entry == nil {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			p.mu.Unlock()
			return nil, nil, nil, err
		}
		if err := watcher.Add(path); err != nil {
			p.mu.Unlock()
			return nil, nil, nil, errors.Join(err, watcher.Close())
		}
		ctx, cancel := context.WithCancel(context.Background())
		entry = &directoryWatch{cancel: cancel, done: make(chan struct{}), subscribers: make(map[chan FileChange]struct{})}
		p.entries[path] = entry
		go p.run(ctx, path, entry, watcher)
	}
	updates := make(chan FileChange, 1)
	entry.subscribers[updates] = struct{}{}
	p.mu.Unlock()
	release := sync.OnceFunc(func() {
		p.mu.Lock()
		delete(entry.subscribers, updates)
		last := len(entry.subscribers) == 0
		if last {
			if p.entries[path] == entry {
				delete(p.entries, path)
			}
			entry.cancel()
		}
		p.mu.Unlock()
		if last {
			<-entry.done
		}
	})
	return entry, updates, release, nil
}

func (p *directoryWatches) publish(entry *directoryWatch, change FileChange) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for updates := range entry.subscribers {
		next := change
		select {
		case updates <- next:
		default:
			select {
			case previous := <-updates:
				next = mergeChanges(previous, next)
			default:
			}
			updates <- next
		}
	}
}

func mergeChanges(before, after FileChange) FileChange {
	result := FileChange{Refresh: before.Refresh || after.Refresh, Rescan: before.Rescan || after.Rescan}
	if result.Rescan {
		return result
	}
	names := make(map[string]struct{})
	for _, name := range append(append([]string(nil), before.Names...), after.Names...) {
		names[name] = struct{}{}
	}
	if len(names) > 128 {
		result.Rescan = true
		return result
	}
	for name := range names {
		result.Names = append(result.Names, name)
	}
	sort.Strings(result.Names)
	return result
}

func (p *directoryWatches) run(ctx context.Context, path string, entry *directoryWatch, watcher *fsnotify.Watcher) {
	defer close(entry.done)
	defer watcher.Close()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	change := FileChange{}
	names := make(map[string]struct{})
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			previous := change.Refresh
			relative, err := filepath.Rel(path, event.Name)
			if err != nil || relative == "." || !filepath.IsLocal(relative) || filepath.Dir(relative) != "." {
				change.Rescan = true
			} else if !change.Rescan {
				names[relative] = struct{}{}
				if len(names) > 128 {
					change.Rescan = true
					clear(names)
				}
			}
			change.Refresh = true
			if !previous {
				timer.Reset(150 * time.Millisecond)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			if !errors.Is(err, fsnotify.ErrEventOverflow) {
				entry.err = err
				return
			}
			if !change.Refresh {
				timer.Reset(150 * time.Millisecond)
			}
			change.Refresh, change.Rescan = true, true
		case <-timer.C:
			if change.Refresh && !change.Rescan {
				for name := range names {
					change.Names = append(change.Names, name)
				}
				sort.Strings(change.Names)
			}
			p.publish(entry, change)
			if change.Refresh {
				timer.Reset(5 * time.Second)
			} else {
				timer.Reset(time.Second)
			}
			change = FileChange{}
			clear(names)
		}
	}
}
