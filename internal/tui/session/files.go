// Purpose: Handle scoped column browsing, paged directories and persistent path selection.
// Pattern: Presentation handler with explicit filesystem calls.
// Usage: Called by local !dir and file-view key events.
// Related: internal/tui/browser/browser.go, internal/tools/local/references.go.
package session

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

func (a *app) openFiles(path string) {
	if path == "" {
		path = "."
	}
	ref, err := a.opts.Tools.Reference(a.settings, path)
	if err != nil {
		a.fail(err)
		return
	}
	if !ref.Directory {
		a.status = "Not a directory"
		return
	}
	entries, more, err := a.opts.Tools.Directory(a.ctx, a.settings, ref.Path, 0, 1000)
	if err != nil {
		a.fail(err)
		return
	}
	a.browser.Open(ref.Path, entries, more)
	a.page = "files"
}
func (a *app) fileKey(e *tcell.EventKey) {
	switch e.Key() {
	case tcell.KeyTAB:
		a.browser.Input = true
	case tcell.KeyUp:
		a.browser.Move(-1)
	case tcell.KeyDown:
		c := a.browser.Current()
		if c != nil && c.Index == len(c.Entries)-1 && c.More {
			entries, more, err := a.opts.Tools.Directory(a.ctx, a.settings, c.Path, c.Offset+1000, 1000)
			if err != nil {
				a.fail(err)
				return
			}
			c.Offset += 1000
			c.Entries = entries
			c.Index = 0
			c.More = more
		} else {
			a.browser.Move(1)
		}
	case tcell.KeyPgUp:
		c := a.browser.Current()
		if c != nil && c.Offset > 0 {
			offset := max(0, c.Offset-1000)
			entries, more, err := a.opts.Tools.Directory(a.ctx, a.settings, c.Path, offset, 1000)
			if err != nil {
				a.fail(err)
				return
			}
			c.Offset = offset
			c.Entries = entries
			c.Index = 0
			c.More = more
		}
	case tcell.KeyLeft:
		a.browser.Left()
	case tcell.KeyRight, tcell.KeyEnter:
		path, dir := a.browser.Target()
		if path == "" || !dir {
			return
		}
		entries, more, err := a.opts.Tools.Directory(a.ctx, a.settings, path, 0, 1000)
		if err != nil {
			a.fail(err)
			return
		}
		a.browser.Push(path, entries, more)
	case tcell.KeyRune:
		if e.Rune() == ' ' {
			path, _ := a.browser.Target()
			if path == "" {
				return
			}
			ref, err := a.opts.Tools.Reference(a.settings, path)
			if err != nil {
				a.fail(err)
				return
			}
			selected := !a.browser.Selected[ref.Path]
			if err = a.opts.Store.SelectFile(a.ctx, a.opts.Session.ID, ref.Path, selected); err != nil {
				a.fail(err)
				return
			}
			if selected {
				a.browser.Selected[ref.Path] = true
			} else {
				delete(a.browser.Selected, ref.Path)
			}
			a.status = fmt.Sprintf("selected: %d", len(a.browser.Selected))
			return
		}
		a.menuInput.Insert(string(e.Rune()))
		if a.menuInput.String() == ":q" {
			a.page = "chat"
			a.menuInput.Set("")
		}
	}
}
