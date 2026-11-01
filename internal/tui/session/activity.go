// Purpose: Own task interruption and one shared LAT/compaction indicator timer.
// Pattern: Event-loop state transitions.
// Usage: Esc and runtime phase notices update Activity; shutdown disposes the timer.
// Related: internal/tui/session/app.go, internal/tui/session/tasks.go, internal/tui/view/activity.go.
package session

import (
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/tui/view"
)

func (a *app) activity() view.Activity {
	return view.Activity{Language: a.language, Elapsed: a.work.elapsed, TokensPerSecond: a.chat.tokensPerSecond, Running: a.work.state.active(), Compressing: a.work.compressing, LAT: a.chat.latOutput.running, Dots: a.work.indicatorDots, ShowCost: a.connection.selection.Connection.Kind == connection.OpenRouter}
}
func (a *app) interruptTask() {
	if !a.work.state.agent() || !a.work.state.active() || a.work.cancel == nil {
		return
	}
	a.restorePending()
	a.files.cancelDirectory()
	a.work.state.stop()
	a.stopElapsed()
	a.work.cancel()
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
	if a.work.state.stopping() {
		return
	}
	a.work.compressing = true
	a.work.refreshIndicator(a.chat.latOutput.running)
}

func (a *app) stopCompression() {
	a.work.compressing = false
	a.work.refreshIndicator(a.chat.latOutput.running)
}

func (a *app) startLATIndicator() {
	if a.work.state.stopping() {
		return
	}
	a.chat.latOutput.running = true
	a.work.refreshIndicator(a.chat.latOutput.running)
}

func (a *app) stopLATIndicator() {
	a.chat.latOutput.running = false
	a.work.refreshIndicator(a.chat.latOutput.running)
}
