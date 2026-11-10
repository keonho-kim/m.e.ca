// Purpose: Queue follow-up messages in FIFO order and restore undelivered requests on interruption.
// Pattern: Event-loop state transitions.
// Usage: Tab reserves input; worker completion dispatches it; cancellation restores the draft.
// Related: internal/tui/session/keys.go, internal/tui/session/worker_result.go, internal/tui/session/commands.go.
package session

import (
	"github.com/keonho-kim/meca/internal/commands"
	requestinput "github.com/keonho-kim/meca/internal/commands/input"
	"github.com/keonho-kim/meca/internal/tui/composer/queue"
)

func (a *app) reserveInput() {
	command, err := a.inputCommand()
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
	if a.work.state.stopping() {
		a.status = a.language.Text("중지 처리 중입니다. 초안을 유지합니다")
		return
	}
	if err := a.compose.queued.Append(requestinput.Capture(text, a.selectedPaths(), a.compose.images...)); err != nil {
		a.fail(err)
		return
	}
	a.clearDraftImages()
	a.compose.editor.Set("")
	a.closeCommandHints()
	a.status = ""
}

func (a *app) restorePending() {
	a.restorePendingEditor()
	a.compose.images = a.compose.queued.Drain(&a.compose.editor, a.compose.images...)
	a.compose.imageEpoch++
	a.refreshHints()
}

func (a *app) finishPending(success bool) {
	if a.compose.queued.Edit != nil {
		a.compose.queued.Edit.Completion = queue.Stopped
		if success && !a.exit && a.ctx.Err() == nil {
			a.compose.queued.Edit.Completion = queue.Succeeded
		}
		return
	}
	if len(a.compose.queued.Messages) == 0 {
		return
	}
	if !success || a.exit || a.ctx.Err() != nil {
		a.restorePending()
		return
	}
	request, draft := a.compose.queued.Messages[0], a.compose.editor
	a.compose.editor.Set(request.Text)
	a.dispatchRequest(request)
	if !a.work.state.active() {
		a.compose.editor = draft
		a.restorePending()
		return
	}
	a.compose.queued.Shift()
	a.compose.editor = draft
	a.refreshHints()
}
