// Purpose: Render fixed-width file columns with independent vertical and horizontal positions.
// Pattern: Presentation renderer.
// Usage: Called when the file browser owns terminal focus.
// Related: internal/tui/browser/browser.go, internal/tui/session/render.go, internal/tui/view/styles.go.
package session

import (
	"fmt"
	"github.com/keonho-kim/v-code/internal/tui/view"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
)

func (a *app) renderFiles(w, h int) {
	columns := a.browser.Columns
	if len(columns) == 0 {
		return
	}
	width := a.opts.Config.UI.ColumnWidth
	visible := max(1, w/(width+1))
	start := max(0, a.browser.Active-visible+1)
	view.Text(a.screen, 0, 0, w, a.opts.Vault.Redact(columns[a.browser.Active].Path), tcell.StyleDefault)
	for i := start; i < min(len(columns), start+visible); i++ {
		c := columns[i]
		x := (i - start) * (width + 1)
		view.Text(a.screen, x, 2, width, filepath.Base(filepath.Clean(c.Path))+"/", tcell.StyleDefault)
		offset := max(0, c.Index-(h-7)+1)
		for index := offset; index < len(c.Entries) && index-offset < h-7; index++ {
			entry := c.Entries[index]
			name := entry.Name
			path := filepath.Join(c.Path, name)
			if entry.Directory {
				name += "/"
				path += "/"
			}
			prefix := "[ ] "
			if a.browser.Selected[path] {
				prefix = "[x] "
			}
			style := view.SelectionStyle(i == a.browser.Active && index == c.Index)
			view.Text(a.screen, x, 3+index-offset, width, prefix+a.opts.Vault.Redact(name), style)
		}
		if i < start+visible-1 {
			for y := 2; y < h-3; y++ {
				view.Text(a.screen, x+width, y, 1, "│", tcell.StyleDefault)
			}
		}
	}
	view.Text(a.screen, 0, h-3, w, fmt.Sprintf("selected: %d", len(a.browser.Selected)), tcell.StyleDefault)
	if a.status != "" {
		view.Text(a.screen, 0, h-2, w, a.status, tcell.StyleDefault)
	}
	view.Input(a.screen, a.editor, h-1, w, 1)
	if !a.browser.Input {
		a.screen.HideCursor()
	}
}
