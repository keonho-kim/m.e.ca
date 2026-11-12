// Purpose: Keep body snapshots and immutable prefixed windows consistent across edits.
// Pattern: Editor ownership and layout regression tests.
// Usage: go test ./internal/tui/input -run TestSnapshot.
// Related: internal/tui/input/editor.go, internal/tui/input/layout.go.
package input

import "testing"

func TestSnapshotPreservesPublishedWindowAcrossEdits(t *testing.T) {
	var editor Editor
	editor.Set("한글 abc")
	editor.SetPrefix("> [IMAGE:1] ")
	old := editor.Window(80, 6)
	editor.Insert("!")
	if editor.String() != "한글 abc!" || editor.Window(80, 6).Source != "> [IMAGE:1] 한글 abc!" {
		t.Fatal("append changed the body or prefix")
	}
	editor.Backspace()
	editor.Move(-1)
	editor.Insert("x")
	if editor.String() != "한글 abxc" {
		t.Fatal(editor.String())
	}
	editor.SetPrefix("")
	if editor.Window(80, 6).Source != "> 한글 abxc" || old.Source != "> [IMAGE:1] 한글 abc" {
		t.Fatal("a published immutable window changed")
	}
}
