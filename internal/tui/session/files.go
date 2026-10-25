// Purpose: Handle scoped column browsing, paged directories and persistent path selection.
// Pattern: Input routing to cancellable directory and selection workers.
// Usage: Called by local !dir and file-view key events.
// Related: internal/tui/session/directory.go, internal/tui/session/file_selection.go, internal/tui/session/render_files.go.
package session

import (
	"github.com/gdamore/tcell/v3"
)

func (a *app) openFiles(path string) {
	a.stopFileWatch()
	a.closeFilePreview()
	a.fileSendConfirm = false
	a.fileReserve = false
	a.status = ""
	if path == "" {
		path = "."
	}
	a.menuInput.Set("")
	a.page = "files"
	a.requestDirectory("open", path, 0)
}

func (a *app) closeFiles() {
	a.stopFileWatch()
	a.fileSendConfirm = false
	a.fileReserve = false
	a.closeFilePreview()
	a.cancelDirectory()
	a.menuInput.Set("")
	a.browser.Input = false
	a.browser.Columns = nil
	a.browser.Active = 0
	a.page = "chat"
}

func (a *app) fileKey(e *tcell.EventKey) {
	if a.paste {
		return
	}
	if a.directoryLoading {
		if e.Key() == tcell.KeyLeft {
			a.cancelDirectory()
		} else {
			return
		}
	}
	switch e.Key() {
	case tcell.KeyBacktab:
		a.browser.Input = true
	case tcell.KeyTab:
		if e.Modifiers()&tcell.ModShift != 0 {
			a.browser.Input = true
		} else {
			a.openFilePreview(false)
		}
	case tcell.KeyEnter:
		a.openFilePreview(true)
	case tcell.KeyUp:
		a.browser.Move(-1)
	case tcell.KeyDown:
		c := a.browser.Current()
		if c != nil && c.Index == len(c.Entries)-1 && c.More {
			a.requestDirectory("page", c.Path, c.Offset+1000)
		} else {
			a.browser.Move(1)
		}
	case tcell.KeyPgUp:
		c := a.browser.Current()
		if c != nil && c.Offset > 0 {
			a.requestDirectory("page", c.Path, max(0, c.Offset-1000))
		}
	case tcell.KeyLeft:
		a.browser.Left()
	case tcell.KeyRight:
		path, dir := a.browser.Target()
		if path == "" || !dir {
			return
		}
		a.requestDirectory("push", path, 0)
	case tcell.KeyRune:
		if e.Str() == " " {
			path, _ := a.browser.Target()
			if path == "" {
				return
			}
			a.selectFile(path)
			return
		}
	}
}
