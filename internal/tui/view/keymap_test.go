// Purpose: Verify consistent key ordering and atomic labels in narrow keyboard-hint rows.
// Pattern: Pure layout tests.
// Usage: go test ./internal/tui/view -run TestKeymapLayout.
// Related: internal/tui/view/keymap.go.
package view

import (
	"github.com/rivo/uniseg"
	"reflect"
	"testing"
)

func TestKeymapLayoutOrderAndBounds(t *testing.T) {
	hints := []Shortcut{{"↑↓", "선택"}, {"Enter", "실행"}, {"Shift+Enter", "줄바꿈"}, {"Esc", "닫기"}, {"^C", "종료"}, {"!", "명령"}}
	original := append([]Shortcut(nil), hints...)
	for _, width := range []int{80, 40, 24, 12} {
		rows := keymapLines(hints, width, 1)
		if len(rows) != 1 || rows[0][0].Key != "^C" {
			t.Fatalf("%d: %v", width, rows)
		}
		cells := 0
		for i, s := range rows[0] {
			if i > 0 {
				cells += 2
			}
			cells += uniseg.StringWidth(s.Key + " " + s.Label)
		}
		if cells > width {
			t.Fatalf("clipped hint: %d > %d", cells, width)
		}
		if width >= 24 && rows[0][1].Key != "Esc" {
			t.Fatal("dismiss hint omitted")
		}
	}
	if !reflect.DeepEqual(hints, original) {
		t.Fatal("renderer mutated shortcut state")
	}
}
