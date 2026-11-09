// Purpose: Keep dense-newline previews sparse and preserve wrapped diff state at distant offsets.
// Pattern: Bounded layout contract tests.
// Usage: go test ./internal/tui/view -run TestSparsePreview.
// Related: internal/tui/view/preview.go, internal/tui/textlayout/wrap.go.
package view

import (
	"strings"
	"testing"
)

func TestSparsePreviewDenseNewlines(t *testing.T) {
	layout, err := BuildPreview(t.Context(), strings.Repeat("\n", 256<<10), 80, false)
	if err != nil {
		t.Fatal(err)
	}
	if layout.Count() != 1+(256<<10) || len(layout.index) > 1025 {
		t.Fatal(layout.Count(), len(layout.index))
	}
	if rows := layout.Window(250000, 12); len(rows) != 12 {
		t.Fatal(len(rows))
	}
}

func TestSparsePreviewWrappedDiffResume(t *testing.T) {
	source := "@@ -1 +1 @@\n+" + strings.Repeat("x", 4096) + "\ncontext"
	layout, err := BuildPreview(t.Context(), source, 4, true)
	if err != nil {
		t.Fatal(err)
	}
	rows := layout.Window(300, 2)
	if len(rows) != 2 || rows[0].Change != '+' || rows[1].Change != '+' {
		t.Fatal(rows)
	}
}
