// Purpose: Verify indexed cursor/selection positions match complete source scanning.
// Pattern: Differential layout contract tests.
// Usage: go test ./internal/tui/input -run TestIndexedLayout.
// Related: internal/tui/input/layout.go, internal/tui/input/navigation.go.
package input

import (
	"testing"
	"unicode/utf8"
)

func TestIndexedLayoutPositions(t *testing.T) {
	for _, text := range []string{"", "hello\n", "\n\n", "a\tb\n\nend", "한글 👩‍💻 á\n둘째 줄 끝", "🇰🇷🇺🇸\n한"} {
		for _, width := range []int{2, 4, 9, 80} {
			var editor Editor
			editor.Set(text)
			editor.prepareLayout(width)
			for cursor := editor.Len(); cursor >= 0; cursor-- {
				editor.MoveTo(cursor)
				offset := 2 + len(editor.Before())
				wantRow, wantColumn := len(editor.layout.rows)-1, 0
				for i, row := range editor.layout.rows {
					if offset <= row.End && !(offset == row.End && row.Soft) {
						wantRow, wantColumn = i, row.Column(editor.layout.source, offset)
						break
					}
				}
				row, column := editor.position()
				if row != wantRow || column != wantColumn {
					t.Fatalf("%q width=%d cursor=%d: %d/%d want %d/%d", text, width, cursor, row, column, wantRow, wantColumn)
				}
				window := editor.Window(width, 3)
				for i, row := range window.Rows {
					for x := 0; x <= width; x++ {
						for _, after := range []bool{false, true} {
							offset := row.Offset(window.Source, x, after)
							want := max(0, utf8.RuneCountInString(window.Source[:offset])-2)
							if got := window.Offset(x, i, after); got != want {
								t.Fatalf("selection at %d/%d: %d want %d", x, i, got, want)
							}
						}
					}
				}
			}
			editor.Insert("새 글\n")
			editor.MoveVertical(-1, width)
			editor.Window(width, 3)
		}
	}
}
