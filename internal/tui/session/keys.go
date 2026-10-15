// Purpose: Route focused keyboard input into local commands, editor actions and menus.
// Pattern: Event Loop input handler.
// Usage: Called only by the terminal loop for tcell key events.
// Related: internal/tui/session/app.go, internal/tui/session/menus.go, internal/tui/input/editor.go, internal/tui/session/hints.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/commands"
	"github.com/keonho-kim/v-code/internal/tui/input"
)

func (a *app) keyEvent(e *tcell.EventKey) {
	if a.selectionKey(e) {
		return
	}
	a.stopNotice = ""
	if input.ExitKey(e) {
		a.requestExit()
		return
	}
	if a.filePreview != nil && e.Key() == tcell.KeyEscape {
		a.previewKey(e)
		return
	}
	if e.Key() == tcell.KeyEscape && a.agentActive && a.busy {
		a.interruptTask()
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
	if e.Key() == tcell.KeyTab && e.Modifiers()&tcell.ModShift == 0 && !a.commandOpen && a.agentActive && a.busy && !a.paste {
		a.reserveInput()
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
			a.submit()
		}
		return
	case tcell.KeyUp:
		a.recall(-1)
		a.refreshHints()
		return
	case tcell.KeyDown:
		a.recall(1)
		a.refreshHints()
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
	w, h := a.screen.Size()
	a.scroll(n, w, a.chatHeight(w, a.bodyHeight(h)))
}
