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
	display := view.FilesDisplay{Activity: a.activity(), Usage: a.displayTokens(), ColumnWidth: a.opts.Config.UI.ColumnWidth, SelectedCount: len(a.browser.Selected), Status: a.status}
	if a.directoryLoading {
		display.Status = "폴더 읽는 중…"
	}
	display.BrowserWidth = w
	if p := a.filePreview; p != nil {
		x := 0
		if !p.Full {
			display.BrowserWidth = max(1, w/2)
			x = display.BrowserWidth + 1
		}
		rows := p.Rows(max(1, w-x))
		p.Offset = min(p.Offset, max(0, len(rows)-max(1, h-2)))
		display.Preview = &view.FilePreviewDisplay{Path: p.Path, Rows: rows[p.Offset:], Full: p.Full, X: x, Command: a.menuInput.String()}
		display.ColumnWidth = min(display.ColumnWidth, max(1, display.BrowserWidth-1))
	}
	if column := a.browser.Current(); column != nil {
		display.Path = a.opts.Vault.Redact(column.Path)
	}
	visible := max(1, display.BrowserWidth/(display.ColumnWidth+1))
	start := max(0, a.browser.Active-visible+1)
	for i := start; i < min(len(a.browser.Columns), start+visible); i++ {
		column := a.browser.Columns[i]
		offset := max(0, column.Index-(h-7)+1)
		value := view.FileColumn{Title: a.opts.Vault.Redact(filepath.Base(filepath.Clean(column.Path)) + "/"), Selected: column.Index - offset, X: (i - start) * (display.ColumnWidth + 1), Active: i == a.browser.Active, Divider: i < start+visible-1}
		for index := offset; index < len(column.Entries) && index-offset < h-7; index++ {
			entry := column.Entries[index]
			name := entry.Name
			path := filepath.Join(column.Path, name)
			marker := "Ⓕ "
			if entry.Directory {
				name += "/"
				path += "/"
				marker = "Ⓓ "
			}
			prefix := "[ ] "
			if a.browser.Selected[path] {
				prefix = "[x] "
			}
			value.Labels = append(value.Labels, prefix+marker+a.opts.Vault.Redact(name))
		}
		display.Columns = append(display.Columns, value)
	}
	commandOpen := !a.browser.Input && fileCommand(a.menuInput.String())
	editor := &a.editor
	if commandOpen {
		editor = &a.menuInput
	}
	display.Input = editor.Window(display.BrowserWidth, 1)
	display.InputFocused = a.browser.Input || commandOpen
	if a.filePreview == nil {
		a.inputRegion = inputRegion{editor: editor, window: display.Input, y: h - 1, width: display.BrowserWidth, revision: editor.Revision()}
	}
	view.Files(a.screen, display, w, h)
}
