// Purpose: Verify preview calculations never own input and discard obsolete geometry or closed views.
// Pattern: Controlled event-loop worker integration tests.
// Usage: go test -race ./internal/tui/session -run TestPreviewLayout.
// Related: internal/tui/session/preview_layout.go, internal/tui/preview/state.go.
package session

import (
	"context"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/tui/browser/controller"

	"github.com/gdamore/tcell/v3"

	"github.com/keonho-kim/meca/internal/tui/preview"
)

func TestPreviewLayoutLeavesInputResponsive(t *testing.T) {
	a := screenFixture(t)
	a.page = "files"
	a.files.filePreview = &preview.State{Path: "large.md", Content: strings.Repeat("# Heading\n\nParagraph.\n\n", 10000), Full: true}
	a.compose.editor.Set("keep draft")
	a.render()
	if !a.files.reads.Layout.Running() || !a.files.filePreview.NeedsLayout(80) {
		t.Fatal("rendering calculated the layout instead of scheduling it")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	a.files.reads.Wait()
	a.workers.Wait()
	for len(a.events) > 0 {
		a.receive(<-a.events)
	}
	if a.files.filePreview != nil || a.files.previewLayoutDesired.Content != "" || a.compose.editor.String() != "keep draft" {
		t.Fatal("closed preview retained a layout source or lost the draft")
	}
}

func TestPreviewLayoutKeepsLatestGeometry(t *testing.T) {
	a := screenFixture(t)
	a.page = "files"
	a.files.filePreview = &preview.State{Path: "guide.md", Content: "# Current\n\n" + strings.Repeat("long paragraph ", 100), Full: true}
	cancelled := false
	a.files.previewLayoutRevision = 1
	a.files.reads.Layout.Begin(1, func() { cancelled = true })
	a.ensurePreviewLayout(80)
	a.ensurePreviewLayout(40)
	a.screen.SetSize(40, 24)
	if !cancelled {
		t.Fatal("new geometry did not cancel obsolete computation")
	}
	a.receivePreviewLayout(&controller.LayoutResult{Request: controller.LayoutRequest{Revision: 1}, Err: context.Canceled})
	finishVisibleReads(t, a)
	if a.files.filePreview.NeedsLayout(40) || !a.files.filePreview.NeedsLayout(80) {
		t.Fatal("obsolete geometry replaced the latest layout")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "R", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "R", 0))
	finishVisibleReads(t, a)
	if a.files.filePreview.Raw || a.files.filePreview.NeedsLayout(40) {
		t.Fatal("two rapid format changes stranded an unprepared preview")
	}
}
