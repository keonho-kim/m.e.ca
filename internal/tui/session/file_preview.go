// Purpose: Own one active preview read, its latest replacement and half/fullscreen viewer navigation.
// Pattern: Single-reader bulkhead with revision-bound completion.
// Usage: File Tab/Enter opens a preview; Esc cancels it and returns to the existing browser.
// Related: internal/tui/browser/preview.go, internal/tools/local/preview.go, internal/tui/session/files.go.
package session

import (
	"context"
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tools/local"
	"github.com/keonho-kim/meca/internal/tui/browser"
)

type previewRequest struct {
	path     string
	revision int
}

func (a *app) openFilePreview(full bool) {
	path, directory := a.browser.Target()
	if path == "" || directory {
		return
	}
	a.closeFilePreview()
	a.menuInput.Set("")
	a.filePreview = &browser.Preview{Path: a.opts.Vault.Redact(path), Full: full, Loading: true}
	a.filePreviewPath = path
	request := previewRequest{path: path, revision: a.previewRevision}
	if a.previewReads.Submit(request) {
		a.startPreviewRead(request)
	}
}

func (a *app) refreshFilePreview() {
	if a.filePreview == nil || a.filePreviewPath == "" {
		return
	}
	a.previewReads.Stop()
	a.previewRevision++
	a.filePreview.Loading, a.filePreview.Error = true, ""
	request := previewRequest{path: a.filePreviewPath, revision: a.previewRevision}
	if a.previewReads.Submit(request) {
		a.startPreviewRead(request)
	}
}

func (a *app) startPreviewRead(request previewRequest) {
	ctx, cancel := context.WithCancel(a.ctx)
	a.previewReads.Begin(request.revision, cancel)
	settings := a.settings
	executor := a.opts.Tools
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		defer cancel()
		result, err := executor.Preview(ctx, settings, request.path)
		if ctx.Err() != nil {
			result, err = local.FilePreview{}, ctx.Err()
		}
		a.post(a.ctx, notice{PreviewRevision: request.revision, FilePreview: &result, Err: err})
	}()
}
func (a *app) closeFilePreview() {
	a.previewReads.Stop()
	a.previewRevision++
	a.filePreview = nil
	a.filePreviewPath = ""
}
func (a *app) receiveFilePreview(n notice) {
	if next := a.previewReads.Complete(n.PreviewRevision); next != nil && !a.exit && a.ctx.Err() == nil {
		a.startPreviewRead(*next)
	}
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
	if previewFormatKey(e) {
		p.ToggleFormat()
		return
	}
	w, h := a.screen.Size()
	h = a.bodyHeight(h)
	if !p.Full {
		w -= w/2 + 1
	}
	maxOffset := max(0, p.Count(w)-max(1, h-2))
	switch e.Key() {
	case tcell.KeyTab:
		if e.Modifiers() == 0 {
			p.Full = false
		}
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
