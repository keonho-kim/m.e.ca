// Purpose: Route image paste and remove the last attachment at the start of the composer.
// Pattern: Composer keyboard adapter.
// Usage: keyEvent calls imageKey before queued-editor handling.
// Related: internal/tui/session/image_paste.go, internal/tui/session/keys.go.
package session

import (
	"runtime"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/agent/media"
)

func imagePasteKey(e *tcell.EventKey, goos string) bool {
	mods := e.Modifiers()
	return e.Key() == tcell.KeyRune && (e.Str() == "v" || e.Str() == "V") && mods == tcell.ModAlt || goos == "darwin" && e.Key() == tcell.KeyCtrlV && (mods == 0 || mods == tcell.ModCtrl)
}

func (a *app) imageKey(e *tcell.EventKey) bool {
	if !a.focus().composer() {
		return false
	}
	if imagePasteKey(e, runtime.GOOS) {
		a.pasteImage()
		return true
	}
	if e.Key() != tcell.KeyBackspace || len(a.compose.images) == 0 {
		return false
	}
	modifiers := e.Modifiers()
	if modifiers != tcell.ModAlt && (modifiers != 0 || a.compose.editor.Cursor != 0) {
		return false
	}
	last := len(a.compose.images) - 1
	a.compose.images[last] = media.Image{}
	a.compose.images = a.compose.images[:last]
	if last == 0 {
		a.compose.images = nil
	}
	return true
}
