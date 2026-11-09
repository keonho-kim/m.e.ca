// Purpose: Attach clipboard images asynchronously and keep ownership with the current composer draft.
// Pattern: Worker result followed by event-loop-only mutation.
// Usage: Alt+V and macOS Ctrl+V paste; Alt+Backspace removes the last draft image.
// Related: internal/tui/clipboard/image.go, internal/tui/session/tasks.go, internal/tui/composer/queue/state.go.
package session

import (
	"context"
	"time"

	"github.com/keonho-kim/meca/internal/agent/media"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/tui/clipboard"
)

type imagePasteNotice struct {
	Image media.Image
	Epoch uint64
	Err   error
}

func (a *app) pasteImage() {
	if a.imagePasteBusy {
		return
	}
	a.imagePasteBusy = true
	epoch, read := a.compose.imageEpoch, a.opts.ReadImage
	if read == nil {
		read = clipboard.ReadImage
	}
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
		defer cancel()
		image, err := read(ctx)
		_ = a.post(a.ctx, notice{ImagePaste: &imagePasteNotice{Image: image, Epoch: epoch, Err: err}})
	}()
}

func (a *app) receiveImagePaste(n *imagePasteNotice) {
	a.imagePasteBusy = false
	if n.Err != nil {
		a.fail(n.Err)
		return
	}
	if n.Epoch != a.compose.imageEpoch {
		a.stopNotice = a.language.Text("Draft changed; paste the image again")
		return
	}
	images := append(a.composerImages(), n.Image)
	if err := media.CheckInput(images); err != nil {
		a.fail(err)
		return
	}
	a.compose.images = append(a.compose.images, n.Image)
	a.stopNotice = a.language.Text(diagcatalog.Text(diagcatalog.ClipboardImagePasted))
}

// composerImages includes saved and queued drafts, so interruption can restore
// them without creating an over-limit combined request.
func (a *app) composerImages() []media.Image {
	images := append([]media.Image(nil), a.compose.images...)
	for i, request := range a.compose.queued.Messages {
		if a.compose.queued.Edit == nil || i != a.compose.queued.Edit.Index {
			images = append(images, request.Images...)
		}
	}
	if edit := a.compose.queued.Edit; edit != nil {
		images = append(images, edit.DraftImages...)
	}
	return images
}

func (a *app) clearDraftImages() { a.compose.images = nil; a.compose.imageEpoch++ }
