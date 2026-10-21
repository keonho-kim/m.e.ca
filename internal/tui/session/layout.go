// Purpose: Reserve the measured footer consistently for rendering, scrolling and selection.
// Pattern: Presentation layout projection.
// Usage: Render takes one footer snapshot; event handlers query bodyHeight for the same geometry.
// Related: internal/tui/session/render.go, internal/tui/session/keymap.go, internal/tui/view/footer.go.
package session

import "github.com/keonho-kim/v-code/internal/tui/view"

func (a *app) footerLayout(width, height int) (int, []view.Shortcut) {
	if a.textSelection.Dragging() && a.footerRows > 0 {
		// Keep mouse coordinates stable until the drag finishes.
		rows := min(a.footerRows, max(1, height-2))
		return max(0, height-rows), a.shortcuts()
	}
	keys := a.shortcuts()
	body := max(0, height-view.FooterRows(keys, width, height))
	return body, keys
}

func (a *app) bodyHeight(height int) int {
	width, _ := a.screen.Size()
	body, _ := a.footerLayout(width, height)
	return body
}
