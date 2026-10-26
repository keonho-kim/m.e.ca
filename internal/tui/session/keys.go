// Purpose: Route focused keyboard input into local commands, editor actions and menus.
// Pattern: Event Loop input handler.
// Usage: Called only by the terminal loop for tcell key events.
// Related: internal/tui/session/app.go, internal/tui/session/menus.go, internal/tui/input/editor.go, internal/tui/session/hints.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/commands"
	"github.com/keonho-kim/meca/internal/tui/input"
)

func (a *app) keyEvent(e *tcell.EventKey) {
	if !e.Pressed() {
		return
	}
	a.scrollbar.dragging = false
	if a.paste {
		a.pasteKey(e)
		return
	}
	e = input.NormalizeKey(e)
	if a.pendingDeleteKey(e) || a.selectionKey(e) {
		return
	}
	a.stopNotice = ""
	if input.ExitKey(e) {
		a.requestExit()
		return
	}
	if a.pendingEditKey(e) {
		return
	}
	if a.page == "files" && a.fileSendConfirm {
		a.fileSendKey(e)
		return
	}
	if e.Key() == tcell.KeyEnter && !a.paste && e.Modifiers()&tcell.ModShift == 0 {
		text := a.menuInput.String()
		if a.page == "chat" || a.page == "files" && a.browser.Input {
			text = a.editor.String()
		}
		command, err := commands.Parse(text)
		if err == nil && (command.Kind == commands.Exit || command.Kind == commands.NewSession) {
			a.newSession = command.Kind == commands.NewSession
			a.requestExit()
			return
		}
	}
	if a.page == "model" {
		a.modelSettingsKey(e)
		return
	}
	if a.page == "language" {
		a.languageKey(e)
		return
	}
	if a.page == "permission" {
		a.permissionKey(e)
		return
	}
	if e.Key() == tcell.KeyCtrlO && a.filePreview == nil && (a.page == "chat" || a.page == "files" && a.browser.Input) {
		if !a.output.ToggleDetails() {
			a.stopNotice = a.language.Text("접거나 펼칠 도구 결과가 없습니다")
		} else if a.output.Collapsed {
			a.stopNotice = a.language.Text("도구 결과 접음 · Ctrl+O로 펼치기")
		} else {
			a.stopNotice = a.language.Text("도구 결과 펼침 · PgUp/Dn으로 스크롤")
		}
		return
	}
	if a.filePreview != nil && e.Key() == tcell.KeyEscape {
		a.previewKey(e)
		return
	}
	if e.Key() == tcell.KeyEscape {
		if a.page == "files" {
			a.closeFiles()
			return
		}
		if a.commandOpen {
			a.closeCommandHints()
			return
		}
		if a.directoryLoading {
			a.cancelDirectory()
			return
		}
		if a.fileSubmission != nil {
			a.fileSubmission = nil
			a.status = ""
			return
		}
		if a.page == "approval" {
			a.decide(false)
			return
		}
		if a.worker.active() && !a.worker.agent() {
			a.interruptSetup()
			return
		}
		if a.page == "chat" {
			if a.worker.agent() && a.worker.active() {
				a.interruptTask()
				return
			}
		}
	}
	if a.page == "files" && a.filePreview != nil {
		a.previewKey(e)
		return
	}
	if a.page == "files" && !a.browser.Input {
		a.fileKey(e)
		return
	}
	if a.page != "chat" && a.page != "files" {
		a.menuKey(e)
		return
	}
	if (e.Key() == tcell.KeyUp || e.Key() == tcell.KeyDown) && (e.Modifiers() == tcell.ModCtrl || e.Modifiers() == tcell.ModMeta) {
		delta := -1
		if e.Key() == tcell.KeyDown {
			delta = 1
		}
		a.recall(delta)
		a.refreshHints()
		return
	}
	if e.Key() == tcell.KeyTab && e.Modifiers()&tcell.ModShift == 0 && !a.commandOpen && a.worker.agent() && a.worker.active() && !a.paste {
		if a.page == "files" {
			a.confirmFileInput(true)
		} else {
			a.reserveInput()
		}
		return
	}
	if a.page == "files" && (e.Key() == tcell.KeyBacktab || e.Key() == tcell.KeyTab && e.Modifiers()&tcell.ModShift != 0) {
		a.browser.Input = false
		return
	}
	if a.hintKey(e) {
		return
	}
	switch e.Key() {
	case tcell.KeyEnter:
		if a.paste || e.Modifiers()&tcell.ModShift != 0 {
			a.editor.Insert("\n")
			a.refreshHints()
		} else {
			if a.page == "files" {
				a.confirmFileInput(false)
			} else {
				a.submit()
			}
		}
		return
	case tcell.KeyUp:
		w, _ := a.screen.Size()
		a.editor.MoveVertical(-1, w)
		return
	case tcell.KeyDown:
		w, _ := a.screen.Size()
		a.editor.MoveVertical(1, w)
		return
	case tcell.KeyPgUp:
		w, h := a.screen.Size()
		a.scroll(-max(1, a.chatHeight(w, a.bodyHeight(h))), w, a.chatHeight(w, a.bodyHeight(h)))
		return
	case tcell.KeyPgDn:
		w, h := a.screen.Size()
		a.scroll(max(1, a.chatHeight(w, a.bodyHeight(h))), w, a.chatHeight(w, a.bodyHeight(h)))
		return
	}
	input.EditKey(&a.editor, e)
	a.refreshHints()
}
func (a *app) recall(n int) {
	if len(a.inputs) == 0 {
		return
	}
	if a.historyIndex == len(a.inputs) {
		a.draft = a.editor.String()
	}
	a.historyIndex = max(0, min(len(a.inputs), a.historyIndex+n))
	if a.historyIndex == len(a.inputs) {
		a.editor.Set(a.draft)
	} else {
		a.editor.Set(a.inputs[a.historyIndex])
	}
}

func (a *app) mouse(e *tcell.EventMouse) {
	if a.scrollbarMouse(e) {
		return
	}
	if a.selectionMouse(e) {
		return
	}
	n := 0
	if e.Buttons()&tcell.WheelUp != 0 {
		n = -3
	}
	if e.Buttons()&tcell.WheelDown != 0 {
		n = 3
	}
	if n == 0 {
		return
	}
	if a.page == "files" && a.filePreview != nil {
		a.filePreview.Offset = max(0, a.filePreview.Offset+n)
		return
	}
	if a.page == "files" {
		a.browser.Move(n)
		return
	}
	if a.page != "chat" {
		return
	}
	w, h := a.screen.Size()
	a.scroll(n, w, a.chatHeight(w, a.bodyHeight(h)))
}
