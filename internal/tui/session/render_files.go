// Purpose: Project browser selection and sanitized file labels into a display snapshot.
// Pattern: Presentation projection.
// Usage: Session supplies view.Files without transferring browser state.
// Related: internal/tui/browser/browser.go, internal/tui/view/files.go.
package session

import (
	"github.com/keonho-kim/v-code/internal/tui/view"
	"path/filepath"
)

func (a *app) renderFiles(w, h int) {
	if len(a.browser.Columns) == 0 {
		return
	}
	display := view.FilesDisplay{Usage: a.displayTokens(), ColumnWidth: a.opts.Config.UI.ColumnWidth, SelectedCount: len(a.browser.Selected), Status: a.status, Editor: a.editor}
	display.Path = a.opts.Vault.Redact(a.browser.Columns[a.browser.Active].Path)
	visible := max(1, w/(display.ColumnWidth+1))
	start := max(0, a.browser.Active-visible+1)
	for i := start; i < min(len(a.browser.Columns), start+visible); i++ {
		column := a.browser.Columns[i]
		offset := max(0, column.Index-(h-7)+1)
		value := view.FileColumn{Title: a.opts.Vault.Redact(filepath.Base(filepath.Clean(column.Path)) + "/"), Selected: column.Index - offset, X: (i - start) * (display.ColumnWidth + 1), Active: i == a.browser.Active, Divider: i < start+visible-1}
		for index := offset; index < len(column.Entries) && index-offset < h-7; index++ {
			entry := column.Entries[index]
			name := entry.Name
			path := filepath.Join(column.Path, name)
			if entry.Directory {
				name += "/"
				path += "/"
			}
			prefix := "[ ] "
			if a.browser.Selected[path] {
				prefix = "[x] "
			}
			value.Labels = append(value.Labels, prefix+a.opts.Vault.Redact(name))
		}
		display.Columns = append(display.Columns, value)
	}
	commandOpen := !a.browser.Input && fileCommand(a.menuInput.String())
	if commandOpen {
		display.Editor = a.menuInput
	}
	display.InputFocused = a.browser.Input || commandOpen
	view.Files(a.screen, display, w, h)
}
