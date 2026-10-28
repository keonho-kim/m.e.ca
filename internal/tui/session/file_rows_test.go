// Purpose: Verify plain file labels distinguish directories with dotted underlines without changing selection.
// Pattern: Virtual terminal presentation contract tests.
// Usage: go test ./internal/tui/session -run TestFileRows.
// Related: internal/tui/session/render_files.go, internal/tui/view/files.go.
package session

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/fileview"
	"github.com/rivo/uniseg"
)

func TestFileRowsPlainNamesAndDirectoryUnderline(t *testing.T) {
	a := fixture(t)
	a.page = "files"
	a.browser.Open(".", []fileview.Entry{{Name: "src", Directory: true}, {Name: "server.go"}, {Name: "문서", Directory: true}, {Name: "설명.md"}}, false)
	a.browser.Selected["src/"] = true
	a.browser.Selected["설명.md"] = true
	for _, focus := range []int{0, 1} {
		a.browser.Current().Index = focus
		a.render()
		for i, want := range []string{"[x] src/", "[ ] server.go", "[ ] 문서/", "[x] 설명.md"} {
			y := 3 + i
			if got := strings.TrimSpace(screenLine(a.screen, y, a.opts.Config.UI.ColumnWidth)); got != want {
				t.Fatalf("row %d: %q; want %q", i, got, want)
			}
			for x := 0; x < uniseg.StringWidth(want); {
				_, style, cells := a.screen.Get(x, y)
				underline := tcell.UnderlineStyleNone
				if x >= 4 && (i == 0 || i == 2) {
					underline = tcell.UnderlineStyleDotted
				}
				if style.GetUnderlineStyle() != underline {
					t.Fatalf("row %d column %d: underline=%v; want %v", i, x, style.GetUnderlineStyle(), underline)
				}
				if style.HasBold() != (i == focus) || style.HasDim() != (i != focus) {
					t.Fatalf("row %d column %d: cursor emphasis lost", i, x)
				}
				x += max(1, cells)
			}
			_, after, _ := a.screen.Get(uniseg.StringWidth(want), y)
			if after.HasUnderline() {
				t.Fatalf("row %d: underline extended beyond the directory name", i)
			}
		}
		assertForegroundOnly(t, a.screen)
	}
}
