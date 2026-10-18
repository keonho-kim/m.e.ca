// Purpose: Verify editing treats displayed graphemes as indivisible characters.
// Pattern: Behavioral tests.
// Usage: go test ./internal/tui/input.
// Related: internal/tui/input/editor.go, internal/tui/input/navigation.go.
package input

import "testing"

func TestGraphemeEditing(t *testing.T) {
	for _, value := range []string{"a\u0301", "👩‍💻", "🇰🇷", "한"} {
		var e Editor
		e.Set("x" + value)
		e.Backspace()
		if e.String() != "x" {
			t.Fatalf("backspace split %q: %q", value, e.String())
		}
		e.Set(value + "x")
		e.Home()
		e.Delete()
		if e.String() != "x" {
			t.Fatalf("delete split %q: %q", value, e.String())
		}
		e.Set("x" + value)
		e.Move(-1)
		e.Insert("!")
		if e.String() != "x!"+value {
			t.Fatalf("cursor split %q: %q", value, e.String())
		}
	}
}
