// Purpose: Keep asynchronous image paste, queue editing and interrupted draft restoration lossless.
// Pattern: Event-loop and owned-worker contract tests with a fake clipboard.
// Usage: go test -race ./internal/tui/session -run TestImagePaste.
// Related: internal/tui/session/image_paste.go, internal/tui/composer/queue/state.go.
package session

import (
	"context"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
	"github.com/keonho-kim/meca/internal/agent/media"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestImagePasteDraftQueueAndRestore(t *testing.T) {
	a := screenFixture(t)
	image, _ := media.New("clipboard.png", "image/png", "png", 64, 32, []byte("compressed image"))
	a.opts.ReadImage = func(context.Context) (media.Image, error) { return image, nil }
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "v", tcell.ModAlt))
	a.receive(<-a.events)
	a.workers.Wait()
	if len(a.compose.images) != 1 || a.compose.editor.String() != "" {
		t.Fatal("image paste changed text or lost pixels")
	}
	a.render()
	if got := screenLine(a.screen, a.inputRegion.y, 80); !strings.Contains(got, "[IMAGE:1]") {
		t.Fatal("image token not visible inside the input")
	}
	a.reserveMessage("")
	if len(a.compose.images) != 0 || len(a.compose.queued.Messages) != 1 || len(a.compose.queued.Messages[0].Images) != 1 {
		t.Fatal("queue lost image-only request")
	}
	a.navigatePending(-1)
	if !a.compose.queued.Save("", a.compose.images...) {
		t.Fatal("image-only request cannot be saved")
	}
	a.restorePending()
	if len(a.compose.images) != 1 || len(a.compose.queued.Messages) != 0 {
		t.Fatal("interruption lost attachment")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyBackspace2, "", 0))
	if len(a.compose.images) != 0 {
		t.Fatal("image remove retained bytes")
	}
}

func TestImagePasteRejectsStaleDraftAndUsesPlatformShortcut(t *testing.T) {
	a := screenFixture(t)
	image, _ := media.New("clipboard.png", "image/png", "png", 1, 1, []byte("snapshot"))
	a.clearDraftImages()
	a.receiveImagePaste(&imagePasteNotice{Image: image, Epoch: 0})
	if len(a.compose.images) != 0 {
		t.Fatal("stale paste crossed into a different draft")
	}
	key := tcell.NewEventKey(tcell.KeyCtrlV, "", 0)
	if !imagePasteKey(key, "darwin") || imagePasteKey(key, "linux") {
		t.Fatal("incorrect platform shortcut")
	}
}

func TestImagePasteHintsFollowModelCapability(t *testing.T) {
	a := screenFixture(t)
	if len(a.appendImageShortcuts(nil)) != 0 {
		t.Fatal("unavailable image hint changed the base footer")
	}
	a.connection.selection.Details = &connection.ModelInfo{ImageInput: connection.DeclaredImageInput(true, "fixture")}
	if keys := a.appendImageShortcuts(nil); len(keys) != 1 || keys[0].Label != "Paste image" {
		t.Fatal("vision model has no paste hint")
	}
	a.connection.selection.Details = nil
	image, _ := media.New("draft.png", "image/png", "png", 1, 1, []byte("image"))
	a.compose.images = []media.Image{image}
	if keys := a.appendImageShortcuts(nil); len(keys) != 1 || keys[0].Key != "Backspace" || keys[0].Label != "Remove image" {
		t.Fatal("draft attachment cannot be removed before model selection")
	}
}

func TestImageTokensStayBlueAndOutOfEditableText(t *testing.T) {
	a := screenFixture(t)
	image, _ := media.New("screen.png", "image/png", "png", 2, 1, []byte("pixels"))
	a.compose.images = []media.Image{image, image}
	a.compose.editor.Set("한글 caption")
	a.render()
	row := a.inputRegion.y
	if got := screenLine(a.screen, row, 80); !strings.Contains(got, "> [IMAGE:1] [IMAGE:2] 한글 caption") {
		t.Fatal(got)
	}
	_, style, _ := a.screen.Get(2, row)
	foreground := style.GetForeground()
	if foreground != color.DodgerBlue {
		t.Fatal("image token lost its blue style")
	}
	if a.compose.editor.String() != "한글 caption" {
		t.Fatal("image token entered request text")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyBackspace, "", 0))
	if len(a.compose.images) != 2 || a.compose.editor.String() != "한글 captio" {
		t.Fatal("backspace within the caption did not edit the caption")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyHome, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyBackspace, "", 0))
	if len(a.compose.images) != 1 || a.compose.editor.String() != "한글 captio" {
		t.Fatal("backspace at the start of the caption did not remove the last image")
	}
	a.render()
	if got := screenLine(a.screen, a.inputRegion.y, 80); strings.Contains(got, "[IMAGE:2]") || !strings.Contains(got, "[IMAGE:1]") {
		t.Fatal(got)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnd, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyBackspace, "", tcell.ModAlt))
	a.render()
	if got := screenLine(a.screen, a.inputRegion.y, 80); strings.Contains(got, "IMAGE") {
		t.Fatal(got)
	}
}
