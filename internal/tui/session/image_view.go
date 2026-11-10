// Purpose: Project display-only image tokens and actionable image keyboard hints from composer state.
// Pattern: Pure terminal presentation projection.
// Usage: Chat, file and queued composers call these projections during rendering.
// Related: internal/tui/session/render_chat.go, internal/tui/session/keymap.go, internal/tui/session/image_keys.go.
package session

import (
	"runtime"
	"strconv"

	"github.com/keonho-kim/meca/internal/agent/media"
	"github.com/keonho-kim/meca/internal/tui/input"
	"github.com/keonho-kim/meca/internal/tui/view"
)

var imagePrefixes = func() [media.MaxImages + 1]string {
	var prefixes [media.MaxImages + 1]string
	prefixes[0] = "> "
	for i := 1; i < len(prefixes); i++ {
		prefixes[i] = prefixes[i-1] + "[IMAGE:" + strconv.Itoa(i) + "] "
	}
	return prefixes
}()

func (a *app) composerWindow(width, rows int) input.Window {
	a.compose.editor.SetPrefix(imagePrefixes[min(len(a.compose.images), media.MaxImages)])
	return a.compose.editor.Window(width, rows)
}

func (a *app) appendImageShortcuts(keys []view.Shortcut) []view.Shortcut {
	if details := a.connection.selection.Details; details != nil && details.ImageInput.Supported() {
		key := "Alt+V"
		if runtime.GOOS == "darwin" {
			key = "^V"
		}
		keys = append(keys, view.Shortcut{Key: key, Label: "Paste image"})
	}
	if len(a.compose.images) > 0 {
		key := "Backspace"
		if a.compose.editor.Cursor != 0 {
			key = "Alt+Backspace"
		}
		keys = append(keys, view.Shortcut{Key: key, Label: "Remove image"})
	}
	return keys
}
