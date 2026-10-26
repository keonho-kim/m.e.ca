// Purpose: Queue follow-up messages in FIFO order and restore undelivered requests on interruption.
// Pattern: Event-loop state transitions.
// Usage: Tab reserves input; worker completion dispatches it; cancellation restores the draft.
// Related: internal/tui/session/keys.go, internal/tui/session/worker_result.go, internal/tui/session/commands.go.
package session

import (
	"github.com/keonho-kim/meca/internal/commands"
	"github.com/keonho-kim/meca/internal/tui/session/queue"
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
	a.reserveMessage(command.Argument)
}

func (a *app) reserveMessage(text string) {
	if a.worker.stopping() {
		a.status = a.language.Text("중지 처리 중입니다. 초안을 유지합니다")
		return
	}
	a.queued.Messages = append(a.queued.Messages, text)
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
	text, draft := a.queued.Messages[0], a.editor
	if a.page == "files" {
		a.closeFiles()
	}
	a.editor.Set(text)
	a.submitCommand(commands.Command{Kind: commands.Message, Argument: text})
	if !a.worker.active() {
		a.editor = draft
		a.restorePending()
		return
	}
	a.queued.Shift()
	a.editor = draft
	a.refreshHints()
}
