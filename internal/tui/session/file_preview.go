// Purpose: Own one active preview read, its latest replacement and half/fullscreen viewer navigation.
// Pattern: Single-reader bulkhead with revision-bound completion.
// Usage: File Tab/Enter opens a preview; Esc cancels it and returns to the existing browser.
// Related: internal/tui/preview/state.go, internal/fileview/preview.go, internal/tui/session/files.go.
package session

import (
	"github.com/keonho-kim/meca/internal/tui/browser/controller"

	"github.com/gdamore/tcell/v3"

	"github.com/keonho-kim/meca/internal/tui/preview"
)

func (a *app) openFilePreview(full bool) {
	path, directory := a.files.browser.Target()
	if path == "" || directory {
		return
	}
	a.closeFilePreview()
	a.menuInput.Set("")
	a.files.filePreview = &preview.State{Path: a.opts.Vault.Redact(path), Full: full, Loading: true}
	a.files.filePreviewPath = path
	request := controller.PreviewRequest{Path: path, Revision: a.files.previewRevision}
	if a.files.reads.Preview.Submit(request) {
		a.startPreviewRead(request)
	}
}

func (a *app) refreshFilePreview() {
	if a.files.filePreview == nil || a.files.filePreviewPath == "" {
		return
	}
	a.files.reads.Preview.Stop()
	a.stopPreviewLayout()
	a.files.previewRevision++
	a.files.filePreview.Loading, a.files.filePreview.Error = true, ""
	a.files.filePreview.ReleaseLayout()
	request := controller.PreviewRequest{Path: a.files.filePreviewPath, Revision: a.files.previewRevision}
	if a.files.reads.Preview.Submit(request) {
		a.startPreviewRead(request)
	}
}

func (a *app) startPreviewRead(request controller.PreviewRequest) {
	a.files.reads.ReadPreview(a.ctx, a, a.opts.Files, a.settings, request)
}
func (a *app) closeFilePreview() {
	a.files.reads.Preview.Stop()
	a.stopPreviewLayout()
	a.files.previewRevision++
	a.files.filePreview = nil
	a.files.filePreviewPath = ""
}
func (a *app) receiveFilePreview(n notice) {
	if next := a.files.reads.Preview.Complete(n.PreviewRevision); next != nil && !a.exit && a.ctx.Err() == nil {
		a.startPreviewRead(*next)
	}
	a.files.applyPreview(n.PreviewRevision, n.FilePreview, n.Err, a.opts.Vault.Redact)
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
	p := a.files.filePreview
	if previewFormatKey(e) {
		p.ToggleFormat()
		a.stopPreviewLayout()
		return
	}
	if e.Key() == tcell.KeyTab && e.Modifiers() == 0 {
		p.Full = false
		return
	}
	if e.Key() == tcell.KeyEnter {
		p.Full = true
		return
	}
	w, h := a.screen.Size()
	h = a.bodyHeight(h)
	if !p.Full {
		w -= w/2 + 1
	}
	if p.Loading || p.NeedsLayout(w) {
		return
	}
	maxOffset := max(0, p.Count(w)-max(1, h-2))
	switch e.Key() {
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

func previewFormatKey(e *tcell.EventKey) bool {
	if e.Key() != tcell.KeyRune {
		return false
	}
	switch e.Modifiers() {
	case 0:
		// Terminals may report shifted letters as uppercase runes without modifier bits.
		return e.Str() == "R"
	case tcell.ModShift:
		return e.Str() == "R" || e.Str() == "r"
	default:
		return false
	}
}
