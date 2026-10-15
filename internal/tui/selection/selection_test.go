// Purpose: Verify visible text selection preserves Unicode and reverse multiline ranges.
// Pattern: Terminal selection contract tests.
// Usage: go test ./internal/tui/selection.
// Related: internal/tui/selection/selection.go.
package selection

import (
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
	"testing"
)

func TestSelectionUnicodeAndReverse(t *testing.T) {
	screen, err := tcell.NewTerminfoScreenFromTty(vt.NewMockTerm(vt.MockOptSize{X: 20, Y: 5}))
	if err != nil {
		t.Fatal(err)
	}
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.Put(0, 1, "한", tcell.StyleDefault)
	screen.Put(2, 1, "글", tcell.StyleDefault)
	screen.Put(4, 1, "!", tcell.StyleDefault)
	screen.Put(0, 2, "👩‍💻", tcell.StyleDefault)
	screen.Put(2, 2, "x", tcell.StyleDefault)
	var s Selection
	s.Begin(screen, 2, 2, 1, 3)
	s.Move(1, 1)
	s.Release()
	if got := s.Text(); got != "한글!\n👩‍💻x" {
		t.Fatalf("copied %q", got)
	}
	screen.Put(0, 1, "new", tcell.StyleDefault)
	if got := s.Text(); got != "한글!\n👩‍💻x" {
		t.Fatal("stream changed selection")
	}
}
