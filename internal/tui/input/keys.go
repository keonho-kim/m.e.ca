// Purpose: Translate terminal editing keys into grapheme-aware editor operations.
// Pattern: Input Adapter.
// Usage: Session key handlers call EditKey for focused text editors.
// Related: internal/tui/input/editor.go, internal/tui/session/keys.go.
package input

import "github.com/gdamore/tcell/v3"

func EditKey(editor *Editor, e *tcell.EventKey) {
	switch e.Key() {
	case tcell.KeyRune:
		editor.Insert(e.Str())
	case tcell.KeyLeft:
		editor.Move(-1)
	case tcell.KeyRight:
		editor.Move(1)
	case tcell.KeyHome, tcell.KeyCtrlA:
		editor.Home()
	case tcell.KeyEnd, tcell.KeyCtrlE:
		editor.End()
	case tcell.KeyBackspace:
		editor.Backspace()
	case tcell.KeyDelete:
		editor.Delete()
	case tcell.KeyCtrlW:
		editor.DeleteWord()
	case tcell.KeyCtrlU:
		editor.Replace(0, editor.Cursor, "")
	case tcell.KeyCtrlK:
		editor.Replace(editor.Cursor, len(editor.Text), "")
	case tcell.KeyTab:
		editor.Insert("\t")
	}
}
