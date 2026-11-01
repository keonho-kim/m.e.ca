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
	focus := a.focus()
	if focus == focusFileConfirmation {
		a.fileSendKey(e)
		return
	}
	if a.exitCommandKey(e, focus) {
		return
	}
	switch focus {
	case focusApproval:
		a.menuKey(e)
		return
	case focusModel:
		a.modelSettingsKey(e)
		return
	case focusLanguage:
		a.languageKey(e)
		return
	case focusPermission:
		a.permissionKey(e)
		return
	}
	if e.Key() == tcell.KeyCtrlO && focus.composer() {
		a.toggleToolDetails()
		return
	}
	if focus == focusFilePreview && e.Key() == tcell.KeyEscape {
		a.previewKey(e)
		return
	}
	if e.Key() == tcell.KeyEscape && a.escapeKey(focus) {
		return
	}
	switch focus {
	case focusFilePreview:
		a.previewKey(e)
	case focusFileNavigation:
		a.fileKey(e)
	default:
		if focus.composer() {
			a.composerKey(e, focus)
		} else {
			a.menuKey(e)
		}
	}
}

func (a *app) exitCommandKey(e *tcell.EventKey, focus focusKind) bool {
	if e.Key() != tcell.KeyEnter || e.Modifiers()&tcell.ModShift != 0 {
		return false
	}
	text := a.menuInput.String()
	if focus.composer() {
		text = a.compose.editor.String()
	}
	command, err := commands.Parse(text)
	if err != nil || command.Kind != commands.Exit && command.Kind != commands.NewSession {
		return false
	}
	a.newSession = command.Kind == commands.NewSession
	a.requestExit()
	return true
}

func (a *app) escapeKey(focus focusKind) bool {
	if focus == focusFileInput || focus == focusFileNavigation {
		a.closeFiles()
		return true
	}
	if a.compose.commandOpen {
		a.closeCommandHints()
		return true
	}
	if a.files.directoryLoading {
		a.files.cancelDirectory()
		return true
	}
	if a.compose.fileSubmission != nil {
		a.compose.fileSubmission = nil
		a.status = ""
		return true
	}
	if a.work.state.active() && !a.work.state.agent() {
		a.interruptSetup()
		return true
	}
	if focus.page() == "chat" && a.work.state.agent() && a.work.state.active() {
		a.interruptTask()
		return true
	}
	return false
}

func (a *app) toggleToolDetails() {
	if !a.chat.output.ToggleDetails() {
		a.stopNotice = a.language.Text("접거나 펼칠 도구 결과가 없습니다")
	} else if a.chat.output.Collapsed {
		a.stopNotice = a.language.Text("도구 결과 접음 · Ctrl+O로 펼치기")
	} else {
		a.stopNotice = a.language.Text("도구 결과 펼침 · PgUp/Dn으로 스크롤")
	}
}
