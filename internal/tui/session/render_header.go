// Purpose: Refresh status-only changes without rebuilding body layout, selection sources or input.
// Pattern: Incremental presentation adapter using the last full frame's sanitized title.
// Usage: The event loop merges header changes into full pending frames or refreshes this row alone.
// Related: internal/tui/session/render.go, internal/tui/session/event_loop.go, internal/tui/view/tokens.go.
package session

import "github.com/keonho-kim/meca/internal/tui/view"

func (a *app) renderHeader() {
	w, h := a.screen.Size()
	if w < 12 || h < 4 {
		return
	}
	switch a.focus() {
	case focusModel, focusLanguage, focusPermission:
		if h-a.footerRows < view.SettingsMinHeight {
			return
		}
	}
	view.Header(a.screen, a.headerTitle, a.chat.displayTokens(), w, a.activity())
	a.screen.Show()
}
