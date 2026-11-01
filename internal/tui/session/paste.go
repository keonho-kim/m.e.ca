// Purpose: Apply bracketed paste as one literal editor transaction without executing pasted commands.
// Pattern: Event-loop input transaction.
// Usage: Terminal paste boundaries select the target; keys accumulate until the closing boundary.
// Related: internal/tui/session/event_loop.go, internal/tui/session/keys.go, internal/tui/input/editor.go.
package session

import (
	"strings"

	"github.com/gdamore/tcell/v3"

	"github.com/keonho-kim/meca/internal/tui/input"
	modelsettings "github.com/keonho-kim/meca/internal/tui/settings/model"
)

type pasteInput struct {
	editor               *input.Editor
	text                 strings.Builder
	limit                int
	singleLine, overflow bool
}

func (a *app) pasteBoundary(start bool) {
	if start {
		a.clearSelection()
		a.paste = true
		a.pasted = pasteInput{}
		switch {
		case a.page == "chat" || a.page == "files" && a.files.browser.Input && a.files.filePreview == nil && !a.files.fileSendConfirm:
			a.pasted.editor = &a.compose.editor
		case a.page == "model" && a.connection.modelForm != nil && a.connection.modelForm.Editing && !a.connection.modelForm.Confirm:
			a.pasted.editor, a.pasted.singleLine = a.connection.modelForm.Editor(), true
		}
		if a.pasted.editor != nil {
			a.pasted.limit = input.MaxBytes - len(a.pasted.editor.String())
		}
		a.refreshHints()
		return
	}
	if a.paste && a.pasted.editor != nil {
		if a.pasted.overflow {
			a.stopNotice = a.language.Text("붙여넣기 취소: 입력은 최대 16MiB입니다")
		} else {
			a.pasted.editor.Insert(a.pasted.text.String())
			if a.page == "model" && a.connection.modelForm != nil {
				f := a.connection.modelForm
				if f.Row == modelsettings.Credential {
					f.KeyChanged = true
				}
				f.Invalidate(f.Row != modelsettings.Name)
			}
		}
	}
	a.paste = false
	a.pasted = pasteInput{}
	a.refreshHints()
	a.refreshModelHints()
}

func (a *app) pasteKey(e *tcell.EventKey) {
	if a.pasted.editor == nil {
		return
	}
	var text string
	switch e.Key() {
	case tcell.KeyRune:
		text = e.Str()
	case tcell.KeyEnter, tcell.KeyCtrlJ:
		if !a.pasted.singleLine {
			text = "\n"
		}
	case tcell.KeyTab:
		if !a.pasted.singleLine {
			text = "\t"
		}
	}
	if a.pasted.text.Len()+len(text) > a.pasted.limit {
		a.pasted.overflow = true
		return
	}
	a.pasted.text.WriteString(text)
}
