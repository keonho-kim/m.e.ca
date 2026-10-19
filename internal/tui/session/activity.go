// Purpose: Own task interruption and compression timer lifecycle on the terminal event loop.
// Pattern: Event-loop state transitions.
// Usage: Esc and runtime phase notices update Activity; shutdown disposes the timer.
// Related: internal/tui/session/app.go, internal/tui/session/tasks.go, internal/tui/view/activity.go.
package session

import (
	"github.com/keonho-kim/v-code/internal/tui/view"
	"time"
)

func (a *app) activity() view.Activity {
	return view.Activity{Language: a.language, Elapsed: a.elapsed, TokensPerSecond: a.tokensPerSecond, Running: a.busy, Compressing: a.compressionTicks != nil, Dots: a.compressionDots}
}
func (a *app) interruptTask() {
	if !a.agentActive || !a.busy || a.taskCancel == nil {
		return
	}
	a.restorePending()
	a.cancelDirectory()
	a.stopRequested = true
	a.stopElapsed()
	a.taskCancel()
	a.stopCompression()
	a.proposal = nil
	a.page = "chat"
	a.choices = nil
	a.closeCommandHints()
	a.status = ""
	a.stopNotice = a.language.Text("작업을 중지하고 정리하고 있습니다")
}
func (a *app) startCompression() {
	if a.stopRequested || a.compressionTimer != nil {
		return
	}
	a.compressionDots = 1
	a.compressionTimer = time.NewTicker(400 * time.Millisecond)
	a.compressionTicks = a.compressionTimer.C
}

func (a *app) advanceCompression() {
	if a.compressionTicks != nil {
		a.compressionDots = a.compressionDots%3 + 1
	}
}
func (a *app) stopCompression() {
	if a.compressionTimer != nil {
		a.compressionTimer.Stop()
	}
	a.compressionTimer = nil
	a.compressionTicks = nil
	a.compressionDots = 0
}
