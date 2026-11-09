// Purpose: Project attachment metadata and actionable image keyboard hints from composer state.
// Pattern: Pure terminal presentation projection.
// Usage: Chat, file and queued composers call these projections during rendering.
// Related: internal/tui/session/render_chat.go, internal/tui/session/keymap.go, internal/tui/session/image_keys.go.
package session

import (
	"runtime"

	"github.com/keonho-kim/meca/internal/tui/view"
)

func (a *app) composerReferenceLines(width, rows int) []string {
	if len(a.compose.images) == 0 {
		return a.selectedReferenceLines(width, rows)
	}
	lines := make([]string, 0, min(len(a.compose.images), max(1, rows)))
	for i, image := range a.compose.images {
		if len(lines) >= max(1, rows) {
			break
		}
		lines = append(lines, a.language.Format("Image %d · %s · %d×%d · %.1f MiB", i+1, image.Path, image.Width, image.Height, float64(len(image.Data))/(1<<20)))
	}
	return append(lines, a.selectedReferenceLines(width, max(0, rows-len(lines)))...)
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
		keys = append(keys, view.Shortcut{Key: "Alt+Backspace", Label: "Remove image"})
	}
	return keys
}
