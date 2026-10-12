// Purpose: Verify Unicode editing and Markdown reference insertion at the cursor.
// Pattern: Behavioral tests.
// Usage: go test ./internal/tui/input.
// Related: internal/tui/input/editor.go.
package input

import (
	"testing"
)

func TestReferenceInsertion(t *testing.T) {
	var e Editor
	e.Insert("읽기 @src/au 뒤")
	e.Move(-2)
	token, start, ok := e.Reference()
	if !ok || token != "src/au" {
		t.Fatalf("token %q %v", token, ok)
	}
	e.Replace(start, e.Cursor, Link("auth.go", "src/auth.go"))
	if e.String() != "읽기 [auth.go](src/auth.go) 뒤" {
		t.Fatal(e.String())
	}
	e.Home()
	e.Insert("한")
	e.Backspace()
	if e.String() != "읽기 [auth.go](src/auth.go) 뒤" {
		t.Fatal(e.String())
	}
}
