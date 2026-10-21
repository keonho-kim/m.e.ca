// Purpose: Verify primary actions precede supporting editing and reading hints on wrapped footers.
// Pattern: Pure keymap layout contract tests.
// Usage: go test ./internal/tui/view -run TestKeymapGroups.
// Related: internal/tui/view/keymap.go, internal/tui/session/keymap.go.
package view

import (
	"slices"
	"testing"
)

func TestKeymapGroups(t *testing.T) {
	keys := []Shortcut{{"PgUp/Dn", "Scroll"}, {"^U", "Clear before cursor"}, {"↑↓", "Select"}, {"Enter", "Apply"}, {"^C", "Exit"}, {"Esc", "Cancel"}}
	rows := keymapLines(keys, 64, 2)
	if len(rows) != 2 {
		t.Fatalf("expected two rows: %+v", rows)
	}
	for i, want := range [][]string{{"^C", "Esc", "Enter", "↑↓"}, {"^U", "PgUp/Dn"}} {
		var got []string
		for _, hint := range rows[i] {
			got = append(got, hint.Key)
		}
		if !slices.Equal(got, want) {
			t.Fatalf("row %d keys=%v; want %v", i, got, want)
		}
	}
	if rows := keymapLines(keys, 160, 2); len(rows) != 1 {
		t.Fatalf("fitting hints were unnecessarily split: %+v", rows)
	}
	if rows := keymapLines(keys, 64, 1); len(rows) != 1 || rows[0][3].Key != "↑↓" {
		t.Fatalf("short screen lost primary navigation: %+v", rows)
	}
}
