// Purpose: Project queued-message editing status and executable keyboard hints.
// Pattern: Presentation projection.
// Usage: Chat layout reserves the notice row; footer selection uses queued-editor hints.
// Related: internal/tui/session/render_chat.go, internal/tui/session/keymap.go, internal/tui/session/pending_edit.go.
package session

import "github.com/keonho-kim/meca/internal/tui/view"

func (a *app) composerNotice() string {
	if a.pendingEdit == nil {
		return a.stopNotice
	}
	text := a.language.Format("Editing queued request %d / %d", a.pendingEdit.index+1, len(a.pending))
	if a.stopNotice != "" {
		text += " · " + a.stopNotice
	}
	return text
}

func (a *app) pendingEditShortcuts() []view.Shortcut {
	keys := []view.Shortcut{
		{Key: "^C", Label: "Exit"},
		{Key: "Esc", Label: "Cancel edit"},
		{Key: "Ctrl/Cmd+Backspace", Label: "Cancel reservation"},
		{Key: "Enter", Label: "Save edit"},
		{Key: "Shift+Enter", Label: "Newline"},
		{Key: "PgUp/Dn", Label: "Scroll"},
	}
	if !a.pendingEditChanged() {
		keys = append(keys, view.Shortcut{Key: "Shift+↑↓/←→", Label: "Browse queue"})
	}
	if a.output.HasDetails() {
		label := "Fold tools"
		if a.output.Collapsed {
			label = "Expand tools"
		}
		keys = append(keys, view.Shortcut{Key: "^O", Label: label})
	}
	return keys
}
