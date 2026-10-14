// Purpose: Cancel only stalled network waits and pause the watchdog while consumers process data.
// Pattern: Request lifecycle owner.
// Usage: SendOnce transfers the idle watchdog to the returned response body.
// Related: internal/provider/transport/transport.go, internal/provider/httpclient/idle_test.go.
package transport

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

const DefaultIdleTimeout = 3 * time.Minute

type idleWatch struct {
	mu             sync.Mutex
	ctx            context.Context
	cancel         context.CancelCauseFunc
	timer          *time.Timer
	interval       time.Duration
	deadline       time.Time
	active, closed bool
}

func watchIdle(parent context.Context, interval time.Duration) *idleWatch {
	ctx, cancel := context.WithCancelCause(parent)
	w := &idleWatch{ctx: ctx, cancel: cancel, interval: interval}
	w.mu.Lock()
	w.active, w.deadline = true, time.Now().Add(interval)
	w.timer = time.AfterFunc(interval, w.expire)
	w.mu.Unlock()
	return w
}

func (w *idleWatch) expire() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed || !w.active {
		return
	}
	if remaining := time.Until(w.deadline); remaining > 0 {
		w.timer.Reset(remaining)
		return
	}
	w.cancel(fmt.Errorf("provider sent no response data for %s", w.interval))
}

func (w *idleWatch) resume() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.closed {
		w.active, w.deadline = true, time.Now().Add(w.interval)
		w.timer.Reset(w.interval)
	}
}

func (w *idleWatch) pause() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.active = false
	w.timer.Stop()
}

func (w *idleWatch) close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	w.timer.Stop()
	w.cancel(nil)
}

type idleBody struct {
	io.ReadCloser
	watch *idleWatch
}

func (b *idleBody) Read(p []byte) (int, error) {
	b.watch.resume()
	n, err := b.ReadCloser.Read(p)
	b.watch.pause()
	if cause := context.Cause(b.watch.ctx); cause != nil {
		return n, cause
	}
	return n, err
}
func (b *idleBody) Close() error {
	b.watch.close()
	return b.ReadCloser.Close()
}
