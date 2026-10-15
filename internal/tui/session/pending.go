// Purpose: Reserve one follow-up message and dispatch it after the current worker completes.
// Pattern: Event-loop state transitions.
// Usage: Tab reserves input; worker completion dispatches it; cancellation restores the draft.
// Related: internal/tui/session/keys.go, internal/tui/session/worker_result.go, internal/tui/session/commands.go.
package session

import (
	"github.com/keonho-kim/v-code/internal/commands"
)

func (a *app) reserveInput() {
	if a.stopRequested {
		a.status = "중지 처리 중입니다. 초안을 유지합니다"
		return
	}
	command, err := commands.Parse(a.editor.String())
	if err != nil {
		a.fail(err)
		return
	}
	if command.Kind == commands.Empty {
		return
	}
	if command.Kind != commands.Message {
		a.status = "일반 메시지만 예약할 수 있습니다"
		return
	}
	if a.pending != "" {
		a.status = "이미 예약된 요청이 있습니다. 새 초안을 유지합니다"
		return
	}
	a.pending = command.Argument
	a.editor.Set("")
	a.closeCommandHints()
	a.status = "현재 작업이 끝나면 예약한 요청을 실행합니다"
}

func (a *app) restorePending() {
	if a.pending == "" {
		return
	}
	text := a.pending
	if draft := a.editor.String(); draft != "" {
		text += "\n\n" + draft
	}
	a.editor.Set(text)
	a.pending = ""
}

func (a *app) finishPending(success bool) {
	if a.pending == "" {
		return
	}
	if !success || a.exit || a.planMode || a.ctx.Err() != nil {
		a.restorePending()
		return
	}
	text, draft := a.pending, a.editor.String()
	a.pending = ""
	a.editor.Set(text)
	a.submit()
	if !a.busy {
		a.pending = text
		a.editor.Set(draft)
		a.restorePending()
		return
	}
	a.editor.Set(draft)
	a.refreshHints()
}
