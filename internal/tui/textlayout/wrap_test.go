// Purpose: Verify wrapping maps wide and combined characters back to exact source spans.
// Pattern: Pure layout tests.
// Usage: go test ./internal/tui/textlayout.
// Related: internal/tui/textlayout/wrap.go.
package textlayout

import "testing"

func TestWrapSourceMapping(t *testing.T) {
	source := "  a\u0301👩‍💻\tmore  \nnext"
	rows := Wrap(source, 8)
	var restored string
	for _, row := range rows {
		start, end := row.Offset(source, 0, false), row.Offset(source, 100, true)
		restored += source[start:end]
		if !row.Soft && end < len(source) {
			restored += "\n"
		}
	}
	if restored != source {
		t.Fatalf("restored %q", restored)
	}
	if rows[0].Offset(source, 4, false) != len("  a\u0301") || rows[0].Offset(source, 4, true) != len("  a\u0301👩‍💻") {
		t.Fatal("wide emoji was split")
	}
}
