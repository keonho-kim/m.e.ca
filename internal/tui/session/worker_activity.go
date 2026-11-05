// Purpose: Own worker status, cancellation linkage and bounded activity clocks.
// Pattern: Event-loop-owned lifecycle state.
// Usage: Runtime and setup handlers set timing state; the event loop supplies status-clock updates.
// Related: internal/tui/session/elapsed.go, internal/tui/session/activity.go, internal/tui/session/worker_state.go.
package session

import (
	"context"
	"time"
)

type workerActivity struct {
	state          workerState
	cancel         context.CancelFunc
	elapsedStarted time.Time
	elapsed        time.Duration
	indicatorTimer *time.Ticker
	indicatorTicks <-chan time.Time
	indicatorDots  int
	compressing    bool
}

func (w *workerActivity) startElapsed() {
	w.elapsedStarted = time.Now()
	w.elapsed = 0
}

func (w *workerActivity) updateElapsed(now time.Time) bool {
	if w.elapsedStarted.IsZero() {
		return false
	}
	elapsed := max(0, now.Sub(w.elapsedStarted))
	changed := elapsed/time.Second != w.elapsed/time.Second
	w.elapsed = elapsed
	return changed
}

func (w *workerActivity) stopElapsed(now time.Time) bool {
	active := !w.elapsedStarted.IsZero()
	w.updateElapsed(now)
	w.elapsedStarted = time.Time{}
	return active
}

func (w *workerActivity) advanceIndicator() {
	if w.indicatorTicks != nil {
		w.indicatorDots = w.indicatorDots%3 + 1
	}
}

func (w *workerActivity) refreshIndicator() {
	if w.compressing {
		if w.indicatorTimer == nil {
			w.indicatorDots = 1
			w.indicatorTimer = time.NewTicker(400 * time.Millisecond)
			w.indicatorTicks = w.indicatorTimer.C
		}
		return
	}
	if w.indicatorTimer != nil {
		w.indicatorTimer.Stop()
	}
	w.indicatorTimer, w.indicatorTicks, w.indicatorDots = nil, nil, 0
}
