// Purpose: Confirm file-composer requests before returning to chat and dispatching them.
// Pattern: Event-loop input transitions.
// Usage: Enter/Tab asks to send/steer/reserve; y accepts, while N or Esc keeps the draft.
// Related: internal/tui/session/keys.go, internal/tui/session/commands.go, internal/tui/session/render_files.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/commands"
)

type fileSubmission struct {
	command  commands.Command
	revision uint64
	reserve  bool
}

func (a *app) fileInputAction() (prompt, action string) {
	if a.fileReserve {
		return "Queue this request? (y/N)", "Queue follow-up"
	}
	if a.worker.agent() && a.worker.active() && !a.worker.stopping() {
		return "Steer this request? (y/N)", "Steer"
	}
	return "Send this request? (y/N)", "Send"
}

func (a *app) confirmFileInput(reserve bool) {
	command, err := a.inputCommand()
	if err != nil {
		a.fail(err)
		return
	}
	if command.Kind == commands.Message {
		a.fileSendConfirm = true
		a.fileReserve = reserve
		return
	}
	a.submit()
}

func (a *app) fileSendKey(e *tcell.EventKey) {
	switch confirmationKey(e) {
	case confirmAccept:
		command, err := a.inputCommand()
		if err != nil {
			a.fail(err)
			return
		}
		reserve := a.fileReserve
		a.closeFiles()
		a.dispatchFileInput(command, reserve)
	case confirmReject, confirmEdit:
		a.fileSendConfirm = false
		a.fileReserve = false
	}
}

func (a *app) dispatchFileInput(command commands.Command, reserve bool) {
	if !reserve || !a.worker.agent() || !a.worker.active() {
		a.submitCommand(command)
		return
	}
	if a.fileSelectionBusy {
		a.fileSubmission = &fileSubmission{command: command, revision: a.editor.Revision(), reserve: true}
		a.status = a.language.Text("Saving selected paths before sending")
		return
	}
	a.reserveMessage(command.Argument)
}

func (a *app) finishFileSubmission() {
	if a.page == "approval" || a.fileSelectionBusy {
		return
	}
	pending := a.fileSubmission
	a.fileSubmission = nil
	if pending == nil || a.exit || a.ctx.Err() != nil || a.page != "chat" {
		return
	}
	if pending.revision != a.editor.Revision() {
		a.status = a.language.Text("Draft changed; send when ready")
		return
	}
	a.status = ""
	a.dispatchFileInput(pending.command, pending.reserve)
}
