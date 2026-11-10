// Purpose: Retire inactive project runtimes and join workspace shutdown.
// Pattern: Resource owner with a bounded idle working set.
// Usage: Workspaces starts one reaper and Close joins it before disposing projects.
// Related: internal/commands/server/workspaces.go.
package server

import (
	"errors"
	"sort"
	"time"
)

// Four idle projects preserve recent navigation without retaining every opened database/client.
const idleWorkspaces = 4

func (w *Workspaces) reap() {
	defer w.workers.Done()
	timer := time.NewTicker(15 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-timer.C:
			w.retireIdle()
		}
	}
}

func (w *Workspaces) retireIdle() {
	w.mu.Lock()
	var idle []*workspaceEntry
	for id, e := range w.entries {
		if id != w.defaultID && e.refs == 0 && e.ready == nil && e.runtime.Service != nil && !e.runtime.Service.Busy() {
			idle = append(idle, e)
		}
	}
	sort.Slice(idle, func(i, j int) bool { return idle[i].used.Before(idle[j].used) })
	type retired struct {
		entry *workspaceEntry
		close func() error
	}
	var pending []retired
	for _, e := range idle[:max(0, len(idle)-idleWorkspaces)] {
		pending = append(pending, retired{e, e.runtime.Close})
		e.runtime = WorkspaceRuntime{}
		e.ready = make(chan struct{})
	}
	w.mu.Unlock()
	var err error
	for _, value := range pending {
		err = errors.Join(err, value.close())
		w.mu.Lock()
		close(value.entry.ready)
		value.entry.ready = nil
		w.mu.Unlock()
	}
	w.mu.Lock()
	w.closeErr = errors.Join(w.closeErr, err)
	w.mu.Unlock()

}

func (w *Workspaces) Close() error {
	w.mu.Lock()
	if w.closed {
		done := w.closeDone
		w.mu.Unlock()
		<-done
		w.mu.Lock()
		defer w.mu.Unlock()
		return w.closeErr
	}
	w.closed = true
	w.cancel()
	w.mu.Unlock()
	w.workers.Wait()
	w.mu.Lock()
	var closers []func() error
	for _, e := range w.entries {
		if e.runtime.Close != nil {
			closers = append(closers, e.runtime.Close)
			e.runtime = WorkspaceRuntime{}
		}
	}
	w.mu.Unlock()
	var err error
	for _, close := range closers {
		err = errors.Join(err, close())
	}
	w.mu.Lock()
	w.closeErr = errors.Join(w.closeErr, err)
	err = w.closeErr
	close(w.closeDone)
	w.mu.Unlock()
	return err
}
