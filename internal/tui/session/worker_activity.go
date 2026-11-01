// Purpose: Own worker status, cancellation linkage and bounded activity clocks.
// Pattern: Event-loop-owned lifecycle state.
// Usage: Runtime and setup handlers update workerActivity; session polls its timer channels.
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
	elapsedTimer   *time.Ticker
	elapsedTicks   <-chan time.Time
	indicatorTimer *time.Ticker
	indicatorTicks <-chan time.Time
	indicatorDots  int
	compressing    bool
}

func (w *workerActivity) startElapsed() {
	w.stopElapsed(time.Now())
	w.elapsedStarted = time.Now()
	w.elapsed = 0
	w.elapsedTimer = time.NewTicker(time.Second)
	w.elapsedTicks = w.elapsedTimer.C
}

func (w *workerActivity) updateElapsed(now time.Time) bool {
	if w.elapsedTimer == nil {
		return false
	}
	w.elapsed = max(0, now.Sub(w.elapsedStarted))
	return true
}

func (w *workerActivity) stopElapsed(now time.Time) bool {
	active := w.updateElapsed(now)
	if w.elapsedTimer != nil {
		w.elapsedTimer.Stop()
	}
	w.elapsedTimer, w.elapsedTicks = nil, nil
	return active
}

func (w *workerActivity) advanceIndicator() {
	if w.indicatorTicks != nil {
		w.indicatorDots = w.indicatorDots%3 + 1
	}
}

func (w *workerActivity) refreshIndicator(latRunning bool) {
	if w.compressing || latRunning {
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
