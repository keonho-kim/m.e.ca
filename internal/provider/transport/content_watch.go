// Purpose: Stop a live provider connection that sends keepalives without meaningful generation progress.
// Pattern: Response-owned cancellable watchdog.
// Usage: Generation attempts touch on text, thinking, tool arguments or continuation progress.
// Related: internal/provider/httpclient/request.go, internal/provider/transport/idle.go.
package transport

import (
	"context"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"sync"
	"time"
)

type ContentWatch struct {
	mu                  sync.Mutex
	ctx                 context.Context
	cancel              context.CancelCauseFunc
	timer               *time.Timer
	interval, remaining time.Duration
	deadline            time.Time
	active, closed      bool
}

func WatchContent(parent context.Context, interval time.Duration) *ContentWatch {
	ctx, cancel := context.WithCancelCause(parent)
	w := &ContentWatch{ctx: ctx, cancel: cancel, interval: interval, remaining: interval, deadline: time.Now().Add(interval), active: true}
	w.timer = time.AfterFunc(interval, w.expire)
	return w
}
func (w *ContentWatch) Context() context.Context { return w.ctx }
func (w *ContentWatch) expire() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed || !w.active {
		return
	}
	if remaining := time.Until(w.deadline); remaining > 0 {
		w.timer.Reset(remaining)
		return
	}
	w.cancel(faults.New(diagcatalog.DeadlineExceeded, diagcatalog.ProviderNoContentProgress, w.interval))
}
func (w *ContentWatch) Activity() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.closed {
		w.remaining = w.interval
		w.deadline = time.Now().Add(w.interval)
		if w.active {
			w.timer.Reset(w.interval)
		}
	}
}
func (w *ContentWatch) Consume(fn func() error) error {
	w.mu.Lock()
	w.remaining = max(time.Nanosecond, time.Until(w.deadline))
	w.active = false
	w.timer.Stop()
	w.mu.Unlock()
	defer func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		if !w.closed {
			w.active = true
			w.deadline = time.Now().Add(w.remaining)
			w.timer.Reset(w.remaining)
		}
	}()
	return fn()
}
func (w *ContentWatch) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	w.timer.Stop()
	w.cancel(nil)
}
