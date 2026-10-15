// Purpose: Verify Unicode editing and literal @ text.
// Pattern: Behavioral tests.
// Usage: go test ./internal/tui/input.
// Related: internal/tui/input/editor.go.
package input

import (
	"testing"
)

func TestLiteralAtInputAndUnicodeEditing(t *testing.T) {
	var e Editor
	e.Insert("읽기 @src/au 뒤")
	e.Home()
	e.Insert("한")
	e.Backspace()
	if e.String() != "읽기 @src/au 뒤" {
		t.Fatal(e.String())
	}
}
