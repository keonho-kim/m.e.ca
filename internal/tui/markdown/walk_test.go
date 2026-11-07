// Purpose: Keep streamed logical rows identical to full rendering across list, code and blank boundaries.
// Pattern: Differential renderer contract tests.
// Usage: go test ./internal/tui/markdown -run TestWalk.
// Related: internal/tui/markdown/walk.go, internal/tui/markdown/render_rows.go.
package markdown

import (
	"reflect"
	"strings"
	"testing"
)

func TestWalkPreservesRenderedRows(t *testing.T) {
	unit := "# title\n\nparagraph **bold** *italic*\n\n- one\n  - nested\n- [x] task\n\n> quote\n\n```text\n한글\ttext\n\n```\n\n| A | B |\n|---|---|\n| a | b |\n\n"
	for _, text := range []string{unit, strings.Repeat(unit, 100), "- ```\n  code\n\n  ```\n", "- # title\n\n  body\n", "a\n\n\n", "\n\n", ""} {
		for _, width := range []int{8, 80} {
			want := Render(text, width)
			var got []Line
			if !Walk(text, width, func(line Line) bool { got = append(got, line); return true }) {
				t.Fatal("valid source rejected")
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("width=%d source=%q: streamed rows changed", width, text[:min(len(text), 80)])
			}
		}
	}
}
