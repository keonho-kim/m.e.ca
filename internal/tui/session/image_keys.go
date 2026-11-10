// Purpose: Route image paste and removal keys without intercepting ordinary text-paste combinations.
// Pattern: Composer keyboard adapter.
// Usage: keyEvent calls imageKey before queued-editor handling.
// Related: internal/tui/session/image_paste.go, internal/tui/session/keys.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/agent/media"
	"runtime"
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
	if (e.Key() == tcell.KeyBackspace || e.Key() == tcell.KeyBackspace2) && e.Modifiers() == tcell.ModAlt && len(a.compose.images) > 0 {
		i := len(a.compose.images) - 1
		a.compose.images[i] = media.Image{}
		a.compose.images = a.compose.images[:i]
		if i == 0 {
			a.compose.images = nil
		}
		return true
	}
	return false
}
