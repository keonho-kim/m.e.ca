// Purpose: Verify display-only composer prefixes preserve text editing, hit testing and cursor motion.
// Pattern: Deterministic editor presentation contracts.
// Usage: go test ./internal/tui/input -run Prefix.
// Related: internal/tui/input/layout.go, internal/tui/input/navigation.go.
package input

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestDisplayPrefixLeavesEditableTextAndOffsetsUnchanged(t *testing.T) {
	var editor Editor
	editor.Set("한글 abc\nnext")
	revision := editor.Revision()
	editor.SetPrefix("> [IMAGE:1] [IMAGE:2] ")
	window := editor.Window(80, 5)
	if window.Source != "> [IMAGE:1] [IMAGE:2] 한글 abc\nnext" || editor.String() != "한글 abc\nnext" || editor.Revision() != revision {
		t.Fatal("display prefix changed the editable body or revision")
	}
	if window.Offset(5, 0, false) != 0 || window.Offset(26, 0, false) != 2 {
		t.Fatalf("prefix hit testing escaped the body: %d, %d", window.Offset(5, 0, false), window.Offset(26, 0, false))
	}
	editor.SetPrefix("")
	if got := editor.Window(80, 5).Source; got != "> 한글 abc\nnext" {
		t.Fatal(got)
	}
}

func TestWrappedPrefixCursorAndVerticalMotionStayWithinEditableBody(t *testing.T) {
	var editor Editor
	editor.Set("abcdef\nuvwxyz")
	editor.SetPrefix("> [IMAGE:1] [IMAGE:2] ")
	editor.MoveTo(0)
	window := editor.Window(12, 8)
	if window.CursorRow < 1 || window.Offset(window.CursorX, window.CursorRow, false) != 0 {
		t.Fatal("wrapped prefix displaced the editable cursor")
	}
	editor.MoveTo(editor.Len())
	editor.MoveVertical(-1, 12)
	if editor.Cursor < 0 || editor.Cursor > editor.Len() {
		t.Fatal("vertical cursor outside body")
	}
	for _, width := range []int{2, 12, 24, 80} {
		for cursor := 0; cursor <= editor.Len(); cursor++ {
			editor.MoveTo(cursor)
			w := editor.Window(width, 20)
			if got := w.Offset(w.CursorX, w.CursorRow, false); got != cursor {
				t.Fatalf("width %d cursor %d round trip %d", width, cursor, got)
			}
		}
	}
	editor.SetPrefix("> ")
	if strings.Contains(editor.Window(12, 8).Source, "IMAGE") || utf8.RuneCountInString(editor.String()) != editor.Len() {
		t.Fatal("prefix retained")
	}
}

func BenchmarkCachedPrefixedWindow(b *testing.B) {
	var editor Editor
	editor.Set("Check these screenshots")
	editor.SetPrefix("> [IMAGE:1] [IMAGE:2] ")
	editor.Window(120, 8)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		editor.Window(120, 8)
	}
}
