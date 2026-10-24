// Purpose: Own column navigation and per-column selection positions for the file view.
// Pattern: Simple Module (presentation state).
// Usage: Owned by the terminal loop; directory I/O is supplied by the caller.
// Related: internal/tui/session/files.go, internal/tools/local/textedit/edit.go.
package browser

import (
	"github.com/keonho-kim/meca/internal/tools/local"
	"path/filepath"
)

type Column struct {
	Path          string
	Entries       []local.Entry
	Index, Offset int
	More          bool
}
type Browser struct {
	Columns  []Column
	Active   int
	Input    bool
	Selected map[string]bool
}

func (b *Browser) Open(path string, entries []local.Entry, more bool) {
	b.Columns = []Column{{Path: path, Entries: entries, More: more}}
	b.Active = 0
	if b.Selected == nil {
		b.Selected = map[string]bool{}
	}
	b.Input = false
}
func (b *Browser) Current() *Column {
	if len(b.Columns) == 0 {
		return nil
	}
	return &b.Columns[b.Active]
}
func (b *Browser) Move(n int) {
	c := b.Current()
	if c != nil {
		c.Index = max(0, min(len(c.Entries)-1, c.Index+n))
		clear(b.Columns[b.Active+1:])
		b.Columns = b.Columns[:b.Active+1]
	}
}
func (b *Browser) Target() (string, bool) {
	c := b.Current()
	if c == nil || len(c.Entries) == 0 {
		return "", false
	}
	entry := c.Entries[c.Index]
	return filepath.Join(c.Path, entry.Name), entry.Directory
}
func (b *Browser) Push(path string, entries []local.Entry, more bool) {
	clear(b.Columns[b.Active+1:])
	b.Columns = b.Columns[:b.Active+1]
	b.Columns = append(b.Columns, Column{Path: path, Entries: entries, More: more})
	b.Active++
}
func (b *Browser) Left() {
	if b.Active > 0 {
		b.Active--
	}
}
