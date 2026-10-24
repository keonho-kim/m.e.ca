// Purpose: Own cancellable file-preview reads and half/fullscreen viewer navigation.
// Pattern: Event-loop worker lifecycle.
// Usage: File Tab/Enter opens a preview; Esc cancels it and returns to the existing browser.
// Related: internal/tui/browser/preview.go, internal/tools/local/preview.go, internal/tui/session/files.go.
package session

import (
	"context"
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/browser"
)

func (a *app) openFilePreview(full bool) {
	path, directory := a.browser.Target()
	if path == "" || directory {
		return
	}
	a.closeFilePreview()
	a.menuInput.Set("")
	a.filePreview = &browser.Preview{Path: a.opts.Vault.Redact(path), Full: full, Loading: true}
	ctx, cancel := context.WithCancel(a.ctx)
	a.previewCancel = cancel
	revision := a.previewRevision
	settings := a.settings
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		defer cancel()
		result, err := a.opts.Tools.Preview(ctx, settings, path)
		if ctx.Err() == nil {
			a.post(ctx, notice{PreviewRevision: revision, FilePreview: &result, Err: err})
		}
	}()
}
func (a *app) closeFilePreview() {
	if a.previewCancel != nil {
		a.previewCancel()
		a.previewCancel = nil
	}
	a.previewRevision++
	a.filePreview = nil
}
func (a *app) receiveFilePreview(n notice) {
	if a.filePreview == nil || n.PreviewRevision != a.previewRevision {
		return
	}
	a.filePreview.Loading = false
	if n.Err != nil {
		a.filePreview.Error = a.opts.Vault.Redact(n.Err.Error())
		return
	}
	if n.FilePreview != nil {
		a.filePreview.Path = n.FilePreview.Path
		a.filePreview.Content = n.FilePreview.Content
		a.filePreview.Truncated = n.FilePreview.Truncated
	}
}
func (a *app) previewKey(e *tcell.EventKey) {
	if a.paste {
		return
	}
	if e.Key() == tcell.KeyEscape {
		a.closeFilePreview()
		a.menuInput.Set("")
		return
	}
	p := a.filePreview
	w, h := a.screen.Size()
	h = a.bodyHeight(h)
	if !p.Full {
		w -= w/2 + 1
	}
	maxOffset := max(0, p.Count(w)-max(1, h-2))
	switch e.Key() {
	case tcell.KeyTab:
		p.Full = false
	case tcell.KeyEnter:
		p.Full = true
	case tcell.KeyUp:
		p.Offset = max(0, p.Offset-1)
	case tcell.KeyDown:
		p.Offset = min(maxOffset, p.Offset+1)
	case tcell.KeyPgUp:
		p.Offset = max(0, p.Offset-max(1, h-2))
	case tcell.KeyPgDn:
		p.Offset = min(maxOffset, p.Offset+max(1, h-2))
	case tcell.KeyHome:
		p.Offset = 0
	case tcell.KeyEnd:
		p.Offset = maxOffset
	}
}
