// Purpose: Own the live worker elapsed-time ticker and reset it between operations.
// Pattern: Event-loop lifecycle.
// Usage: Task and connection workers start timing; cancellation and completion stop it.
// Related: internal/tui/session/app.go, internal/tui/session/activity.go, internal/tui/session/worker_result.go.
package session

import "time"

func (a *app) startElapsed() {
	a.stopElapsed()
	a.elapsedStarted = time.Now()
	a.elapsed = 0
	a.elapsedTimer = time.NewTicker(time.Second)
	a.elapsedTicks = a.elapsedTimer.C
}

func (a *app) updateElapsed(now time.Time) {
	if a.elapsedTimer != nil {
		a.elapsed = max(0, now.Sub(a.elapsedStarted))
		a.updateCommandActivity()
	}
}

func (a *app) stopElapsed() {
	if a.elapsedTimer != nil {
		a.updateElapsed(time.Now())
		a.elapsedTimer.Stop()
	}
	a.elapsedTimer = nil
	a.elapsedTicks = nil
}
