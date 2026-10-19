// Purpose: Verify terminal Ctrl-key normalization preserves modifiers and event metadata.
// Pattern: Input contract tests.
// Usage: go test ./internal/tui/input -run TestNormalizeKey.
// Related: internal/tui/input/normalize.go.
package input

import (
	"testing"

	"github.com/gdamore/tcell/v3"
)

func TestNormalizeKey(t *testing.T) {
	for _, test := range []struct {
		text string
		mod  tcell.ModMask
		want tcell.Key
	}{
		{"o", tcell.ModCtrl, tcell.KeyCtrlO},
		{"O", tcell.ModRCtrl, tcell.KeyCtrlO},
		{"u", tcell.ModLCtrl, tcell.KeyCtrlU},
		{"o", 0, tcell.KeyRune},
		{"o", tcell.ModMeta, tcell.KeyRune},
		{"o", tcell.ModCtrl | tcell.ModShift, tcell.KeyRune},
		{"o", tcell.ModCtrl | tcell.ModAlt, tcell.KeyRune},
		{"한", tcell.ModCtrl, tcell.KeyRune},
	} {
		original := tcell.NewEventKeyEx(tcell.KeyRune, test.text, test.mod, false, tcell.KeyO, 3)
		normalized := NormalizeKey(original)
		if normalized.Key() != test.want || normalized.Modifiers() != test.mod || normalized.Pressed() || normalized.Physical() != original.Physical() || normalized.Repeat() != 3 {
			t.Fatalf("%s %v: %+v", test.text, test.mod, normalized)
		}
		if test.want == tcell.KeyRune && normalized != original {
			t.Fatal("ordinary text or another shortcut was changed")
		}
	}
}
