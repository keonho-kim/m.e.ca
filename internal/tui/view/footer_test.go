// Purpose: Verify footer height expands only for wrapped keyboard hints and preserves a small body.
// Pattern: Pure terminal layout contract tests.
// Usage: go test ./internal/tui/view -run TestFooterRows.
// Related: internal/tui/view/footer.go, internal/tui/view/keymap.go.
package view

import "testing"

func TestFooterRowsWrapOnlyWhenNeeded(t *testing.T) {
	keys := []Shortcut{{"^C", "Quit"}, {"Esc", "Close"}, {"Enter", "Send"}, {"Tab", "Choose"}}
	for _, test := range []struct{ width, height, rows int }{
		{64, 24, 2},
		{24, 24, 3},
		{24, 4, 2},
		{24, 3, 1},
		{24, 0, 0},
	} {
		if got := FooterRows(keys, test.width, test.height); got != test.rows {
			t.Errorf("%+v: footer rows=%d", test, got)
		}
	}
	rows := keymapLines(keys, 24, 2)
	if len(rows) != 2 {
		t.Fatalf("keyboard hints did not wrap: %+v", rows)
	}
	count := 0
	for _, row := range rows {
		for _, key := range row {
			if key.Key == "…" {
				t.Fatal("hints were omitted even though they fit on two rows")
			}
			count++
		}
	}
	if count != len(keys) {
		t.Fatal("wrapped footer lost a keyboard hint")
	}
	if rows := KeymapRows(keys, 12, 20); rows > 2 {
		t.Fatalf("keyboard hints exceeded two rows: %d", rows)
	}
}
