// Purpose: Verify preview calculations never own input and discard obsolete geometry or closed views.
// Pattern: Controlled event-loop worker integration tests.
// Usage: go test -race ./internal/tui/session -run TestPreviewLayout.
// Related: internal/tui/session/preview_layout.go, internal/tui/preview/state.go.
package session

import (
	"context"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/preview"
)

func TestPreviewLayoutLeavesInputResponsive(t *testing.T) {
	a := fixture(t)
	a.page = "files"
	a.filePreview = &preview.State{Path: "large.md", Content: strings.Repeat("# Heading\n\nParagraph.\n\n", 10000), Full: true}
	a.editor.Set("keep draft")
	a.render()
	if !a.previewLayouts.Running() || !a.filePreview.NeedsLayout(80) {
		t.Fatal("rendering calculated the layout instead of scheduling it")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	a.workers.Wait()
	for len(a.events) > 0 {
		a.receive(<-a.events)
	}
	if a.filePreview != nil || a.previewLayoutDesired.Content != "" || a.editor.String() != "keep draft" {
		t.Fatal("closed preview retained a layout source or lost the draft")
	}
}

func TestPreviewLayoutKeepsLatestGeometry(t *testing.T) {
	a := fixture(t)
	a.page = "files"
	a.filePreview = &preview.State{Path: "guide.md", Content: "# Current\n\n" + strings.Repeat("long paragraph ", 100), Full: true}
	cancelled := false
	a.previewLayoutRevision = 1
	a.previewLayouts.Begin(1, func() { cancelled = true })
	a.ensurePreviewLayout(80)
	a.ensurePreviewLayout(40)
	a.screen.SetSize(40, 24)
	if !cancelled {
		t.Fatal("new geometry did not cancel obsolete computation")
	}
	a.receivePreviewLayout(&previewLayoutResult{request: previewLayoutRequest{revision: 1}, err: context.Canceled})
	finishVisibleReads(t, a)
	if a.filePreview.NeedsLayout(40) || !a.filePreview.NeedsLayout(80) {
		t.Fatal("obsolete geometry replaced the latest layout")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "R", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "R", 0))
	finishVisibleReads(t, a)
	if a.filePreview.Raw || a.filePreview.NeedsLayout(40) {
		t.Fatal("two rapid format changes stranded an unprepared preview")
	}
}
