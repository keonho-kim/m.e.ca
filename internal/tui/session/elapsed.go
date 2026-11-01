// Purpose: Own the live worker elapsed-time ticker and reset it between operations.
// Pattern: Event-loop lifecycle.
// Usage: Task and connection workers start timing; cancellation and completion stop it.
// Related: internal/tui/session/app.go, internal/tui/session/activity.go, internal/tui/session/worker_result.go.
package session

import (
	"time"
)

func (a *app) startElapsed() {
	a.stopElapsed()
	a.work.startElapsed()
}

func (a *app) updateElapsed(now time.Time) {
	if a.work.updateElapsed(now) {
		a.updateCommandActivity()
	}
}

func (a *app) stopElapsed() {
	if a.work.stopElapsed(time.Now()) {
		a.updateCommandActivity()
	}
}
