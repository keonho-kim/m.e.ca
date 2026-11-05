// Purpose: Project monotonic worker elapsed time through the shared terminal status clock.
// Pattern: Event-loop lifecycle.
// Usage: Agent and connection workers start timing; whole-second changes refresh command metadata.
// Related: internal/tui/session/app.go, internal/tui/session/activity.go, internal/tui/session/worker_result.go.
package session

import (
	"time"
)

func (a *app) startElapsed() {
	a.stopElapsed()
	a.work.startElapsed()
}

func (a *app) updateElapsed(now time.Time) bool {
	if a.work.updateElapsed(now) {
		a.updateCommandActivity()
		return true
	}
	return false
}

func (a *app) stopElapsed() {
	if a.work.stopElapsed(time.Now()) {
		a.updateCommandActivity()
	}
}
