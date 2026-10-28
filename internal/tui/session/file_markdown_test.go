// Purpose: Verify Markdown preview styling, viewer-only format toggling and contextual help.
// Pattern: Virtual terminal workflow tests.
// Usage: go test ./internal/tui/session -run 'TestMarkdownFilePreview|TestLiteralFilePreview'.
// Related: internal/tui/session/file_preview.go, internal/tui/view/file_preview.go, internal/tui/browser/preview.go.
package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tools/local"
	"github.com/keonho-kim/meca/internal/tui/browser"
	"github.com/keonho-kim/meca/internal/tui/locale"
)

func TestMarkdownFilePreviewRenderingAndToggle(t *testing.T) {
	for _, full := range []bool{false, true} {
		a := fixture(t)
		a.language = locale.English
		if full {
			a.language = locale.Korean
		}
		content := "# Viewer title\n\n**strong** body\n\n```mermaid\ngraph TD\nA --> B\n```\n"
		if err := os.WriteFile(filepath.Join(a.opts.Session.ProjectRoot, "README.MD"), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		a.openFiles(".")
		finishBrowserIO(t, a)
		key := tcell.KeyTab
		if full {
			key = tcell.KeyEnter
		}
		a.keyEvent(tcell.NewEventKey(key, "", 0))
		finishVisibleReads(t, a)
		a.render()
		shown := strings.Join(screenRows(a, 24-a.footerRows), "\n")
		if !strings.Contains(shown, "Viewer title") || strings.Contains(shown, "# Viewer title") || strings.Contains(shown, "**strong**") || !strings.Contains(shown, "A --> B") {
			t.Fatalf("full=%v: Markdown not rendered or Mermaid source missing: %s", full, shown)
		}
		assertPreviewHeadingBold(t, a, "Viewer title")
		if labels := keymapLabels(t, a); labels["Shift+R"] != a.language.Text("Show raw") {
			t.Fatalf("Markdown preview toggle hint missing: %v", labels)
		}
		if !strings.Contains(footerKeyText(a), "Shift+R") {
			t.Fatal("preview toggle is hidden in the actual footer")
		}
		a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "r", 0))
		if a.filePreview.Raw {
			t.Fatal("unmodified lowercase r toggled the preview")
		}
		for _, toggle := range []*tcell.EventKey{
			tcell.NewEventKey(tcell.KeyRune, "R", 0),
			tcell.NewEventKeyEx(tcell.KeyRune, "R", tcell.ModShift, true, tcell.KeyR, 1),
			tcell.NewEventKeyEx(tcell.KeyRune, "r", tcell.ModShift, true, tcell.KeyR, 1),
		} {
			revision := a.previewRevision
			a.keyEvent(toggle)
			a.render()
			shown = strings.Join(screenRows(a, 24-a.footerRows), "\n")
			if a.filePreview.Full != full || !strings.Contains(shown, "# Viewer title") || !strings.Contains(shown, "**strong**") {
				t.Fatalf("key=%v: format toggle changed pane size or lost raw source: %s", toggle, shown)
			}
			if a.previewRevision != revision || a.previewReads.Running() {
				t.Fatal("format toggle reread the file instead of reusing its content")
			}
			if labels := keymapLabels(t, a); labels["Shift+R"] != a.language.Text("Render") {
				t.Fatal(labels)
			}
			a.keyEvent(toggle)
			a.render()
			assertPreviewHeadingBold(t, a, "Viewer title")
		}
	}
}

func assertPreviewHeadingBold(t *testing.T, a *app, title string) {
	t.Helper()
	w, h := a.screen.Size()
	for y := 2; y < h-a.footerRows; y++ {
		line := screenLine(a.screen, y, w)
		if x := strings.Index(line, title); x >= 0 {
			_, style, _ := a.screen.Get(x, y)
			if style.GetAttributes()&tcell.AttrBold == 0 {
				t.Fatalf("heading style missing at %d,%d: %q", x, y, line)
			}
			return
		}
	}
	t.Fatal("rendered heading was not visible")
}

func TestLiteralFilePreviewHasNoFormatToggle(t *testing.T) {
	a := fixture(t)
	a.page = "files"
	a.filePreview = &browser.Preview{Path: "source.go", Content: "**literal source**", Full: true}
	for _, key := range []*tcell.EventKey{tcell.NewEventKey(tcell.KeyRune, "R", 0), tcell.NewEventKeyEx(tcell.KeyRune, "r", tcell.ModShift, true, tcell.KeyR, 1)} {
		a.keyEvent(key)
		if !a.filePreview.Full || keymapLabels(t, a)["Shift+R"] != "" {
			t.Fatal("format key changed a literal preview or advertised Markdown controls")
		}
	}
}

func TestMarkdownFilePreviewRefreshPreservesModeAndPosition(t *testing.T) {
	a := fixture(t)
	a.page, a.previewRevision = "files", 7
	a.filePreview = &browser.Preview{Path: "guide.md", Content: strings.Repeat("**before**\n", 50), Raw: true, Offset: 8, Loading: true}
	a.render()
	if a.filePreview.Offset != 8 {
		t.Fatal("loading reset the existing read position")
	}
	a.receiveFilePreview(notice{PreviewRevision: 7, FilePreview: &local.FilePreview{Path: "guide.md", Content: "# New heading\n\n" + strings.Repeat("**updated**\n", 50)}})
	a.render()
	shown := strings.Join(screenRows(a, 24-a.footerRows), "\n")
	if !a.filePreview.Raw || a.filePreview.Full || a.filePreview.Offset != 8 || !strings.Contains(shown, "**updated**") || strings.Contains(shown, "before") {
		t.Fatalf("refreshed preview lost mode, source or position: %s", shown)
	}
	a.keyEvent(tcell.NewEventKeyEx(tcell.KeyRune, "R", tcell.ModShift, true, tcell.KeyR, 1))
	a.render()
	assertPreviewHeadingBold(t, a, "New heading")
}

func TestMarkdownFilePreviewShortcutLeavesInputEditable(t *testing.T) {
	a := fixture(t)
	a.editor.Set("draft ")
	a.keyEvent(tcell.NewEventKeyEx(tcell.KeyRune, "R", tcell.ModShift, true, tcell.KeyR, 1))
	if a.editor.String() != "draft R" || a.filePreview != nil {
		t.Fatal("viewer-only format key intercepted ordinary input")
	}
}
