// Purpose: Queue follow-up messages in FIFO order and restore undelivered requests on interruption.
// Pattern: Event-loop state transitions.
// Usage: Tab reserves input; worker completion dispatches it; cancellation restores the draft.
// Related: internal/tui/session/keys.go, internal/tui/session/worker_result.go, internal/tui/session/commands.go.
package session

import (
	"github.com/keonho-kim/meca/internal/commands"
	"github.com/keonho-kim/meca/internal/tui/composer"
	"github.com/keonho-kim/meca/internal/tui/composer/queue"
)

func (a *app) reserveInput() {
	command, err := commands.Parse(a.editor.String())
	if err != nil {
		a.fail(err)
		return
	}
	if command.Kind == commands.Empty {
		return
	}
	if command.Kind != commands.Message {
		a.status = a.language.Text("일반 메시지만 예약할 수 있습니다")
		return
	}
	a.dispatchFileInput(command, true)
}

func (a *app) reserveMessage(text string) {
	if a.worker.stopping() {
		a.status = a.language.Text("중지 처리 중입니다. 초안을 유지합니다")
		return
	}
	a.queued.Messages = append(a.queued.Messages, composer.Capture(text, a.selectedPaths()))
	a.editor.Set("")
	a.closeCommandHints()
	a.status = ""
}

func (a *app) restorePending() {
	a.restorePendingEditor()
	a.queued.Drain(&a.editor)
	a.refreshHints()
}

func (a *app) finishPending(success bool) {
	if a.queued.Edit != nil {
		a.queued.Edit.Completion = queue.Stopped
		if success && !a.exit && a.ctx.Err() == nil {
			a.queued.Edit.Completion = queue.Succeeded
		}
		return
	}
	if len(a.queued.Messages) == 0 {
		return
	}
	if !success || a.exit || a.ctx.Err() != nil {
		a.restorePending()
		return
	}
	request, draft := a.queued.Messages[0], a.editor
	a.editor.Set(request.Text)
	a.dispatchRequest(request)
	if !a.worker.active() {
		a.editor = draft
		a.restorePending()
		return
	}
	a.queued.Shift()
	a.editor = draft
	a.refreshHints()
}
