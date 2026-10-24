// Purpose: Apply bracketed paste as one literal editor transaction without executing pasted commands.
// Pattern: Event-loop input transaction.
// Usage: Terminal paste boundaries select the target; keys accumulate until the closing boundary.
// Related: internal/tui/session/event_loop.go, internal/tui/session/keys.go, internal/tui/input/editor.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/input"
	"strings"
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
		case a.page == "chat" || a.page == "files" && a.browser.Input && a.filePreview == nil && !a.fileSendConfirm:
			a.pasted.editor = &a.editor
		case a.page == "model" && a.modelForm != nil && a.modelForm.editing && !a.modelForm.confirm:
			a.pasted.editor, a.pasted.singleLine = a.modelForm.editor(), true
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
			if a.page == "model" && a.modelForm != nil {
				f := a.modelForm
				if f.row == modelCredential {
					f.keyChanged = true
				}
				f.invalidate(f.row != modelName)
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
