// Purpose: Own task interruption and one shared LAT/compaction indicator timer.
// Pattern: Event-loop state transitions.
// Usage: Esc and runtime phase notices update Activity; shutdown disposes the timer.
// Related: internal/tui/session/app.go, internal/tui/session/tasks.go, internal/tui/view/activity.go.
package session

import (
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/tui/view"
	"time"
)

func (a *app) activity() view.Activity {
	return view.Activity{Language: a.language, Elapsed: a.elapsed, TokensPerSecond: a.tokensPerSecond, Running: a.worker.active(), Compressing: a.compressing, LAT: a.latOutput.running, Dots: a.indicatorDots, ShowCost: a.selection.Connection.Kind == connection.OpenRouter}
}
func (a *app) interruptTask() {
	if !a.worker.agent() || !a.worker.active() || a.taskCancel == nil {
		return
	}
	a.restorePending()
	a.cancelDirectory()
	a.worker.stop()
	a.stopElapsed()
	a.taskCancel()
	a.stopLATIndicator()
	a.stopCompression()
	a.proposal = nil
	a.page = "chat"
	a.choices = nil
	a.closeCommandHints()
	a.status = ""
	a.clearToolActivity()
	a.stopNotice = a.language.Text("작업을 중지하고 정리하고 있습니다")
}
func (a *app) startCompression() {
	if a.worker.stopping() {
		return
	}
	a.compressing = true
	a.refreshIndicator()
}

func (a *app) advanceIndicator() {
	if a.indicatorTicks != nil {
		a.indicatorDots = a.indicatorDots%3 + 1
	}
}
func (a *app) stopCompression() {
	a.compressing = false
	a.refreshIndicator()
}

func (a *app) startLATIndicator() {
	if a.worker.stopping() {
		return
	}
	a.latOutput.running = true
	a.refreshIndicator()
}

func (a *app) stopLATIndicator() {
	a.latOutput.running = false
	a.refreshIndicator()
}

func (a *app) refreshIndicator() {
	if a.compressing || a.latOutput.running {
		if a.indicatorTimer == nil {
			a.indicatorDots = 1
			a.indicatorTimer = time.NewTicker(400 * time.Millisecond)
			a.indicatorTicks = a.indicatorTimer.C
		}
		return
	}
	if a.indicatorTimer != nil {
		a.indicatorTimer.Stop()
	}
	a.indicatorTimer, a.indicatorTicks, a.indicatorDots = nil, nil, 0
}
