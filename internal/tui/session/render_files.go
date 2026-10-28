// Purpose: Project browser positions, selected references and sanitized file entries into a snapshot.
// Pattern: Presentation projection.
// Usage: Session supplies view.Files without transferring browser state.
// Related: internal/tui/browser/browser.go, internal/tui/view/files.go.
package session

import (
	"github.com/keonho-kim/meca/internal/tui/view"
	"path/filepath"
)

func (a *app) renderFiles(w, h int) {
	display := view.FilesDisplay{Activity: a.activity(), Usage: a.displayTokens(), ColumnWidth: a.opts.Config.UI.ColumnWidth, Status: a.status}
	if a.directoryLoading {
		display.Status = a.language.Text("Reading directory…")
	}
	if a.fileSendConfirm {
		prompt, _ := a.fileInputAction()
		display.Status = a.language.Text(prompt)
	}
	display.BrowserWidth = w
	if p := a.filePreview; p != nil {
		x := 0
		if !p.Full {
			display.BrowserWidth = max(1, w/2)
			x = display.BrowserWidth + 1
		}
		offset := 0
		if !p.Loading {
			p.Offset = min(p.Offset, max(0, p.Count(max(1, w-x))-max(1, h-2)))
			offset = p.Offset
		}
		mode := ""
		if p.IsMarkdown() {
			mode = a.language.Text("Rendered")
			if p.Raw {
				mode = a.language.Text("Raw")
			}
		}
		display.Preview = &view.FilePreviewDisplay{Path: p.Path, Mode: mode, Rows: p.Window(max(1, w-x), offset, max(1, h-2)), Full: p.Full, X: x}
		display.ColumnWidth = min(display.ColumnWidth, max(1, display.BrowserWidth-1))
	}
	display.Input = a.editor.Window(display.BrowserWidth, min(a.opts.Config.UI.InputMaxLines, max(1, h-4)))
	display.References = a.selectedReferenceLines(display.BrowserWidth, h-len(display.Input.Rows)-4)
	display.Layout = view.Composer(len(display.Input.Rows), h, view.Candidates{}, display.Status, len(display.References))
	display.InputFocused = a.browser.Input && !a.fileSendConfirm
	entryRows := max(0, display.Layout.ReferenceY-4)
	if column := a.browser.Current(); column != nil {
		display.Path = a.opts.Vault.Redact(column.Path)
	}
	visible := max(1, display.BrowserWidth/(display.ColumnWidth+1))
	start := max(0, a.browser.Active-visible+1)
	for i := start; i < min(len(a.browser.Columns), start+visible); i++ {
		column := a.browser.Columns[i]
		offset := max(0, column.Index-entryRows+1)
		value := view.FileColumn{Title: a.opts.Vault.Redact(filepath.Base(filepath.Clean(column.Path)) + "/"), Selected: column.Index - offset, X: (i - start) * (display.ColumnWidth + 1), Active: i == a.browser.Active, Divider: i < start+visible-1}
		for index := offset; index < len(column.Entries) && index-offset < entryRows; index++ {
			entry := column.Entries[index]
			path := filepath.Join(column.Path, entry.Name)
			if entry.Directory {
				path += "/"
			}
			value.Entries = append(value.Entries, view.FileEntry{Name: a.opts.Vault.Redact(entry.Name), Directory: entry.Directory, Marked: a.browser.Selected[path]})
		}
		display.Columns = append(display.Columns, value)
	}
	if a.filePreview == nil && !a.fileSendConfirm {
		a.inputRegion = inputRegion{editor: &a.editor, window: display.Input, y: display.Layout.InputY, width: display.BrowserWidth, revision: a.editor.Revision()}
	}
	view.Files(a.screen, display, w, h)
}
