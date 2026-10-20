// Purpose: Verify event-relative scroll anchors survive appends and reflow.
// Pattern: Presentation behavior tests.
// Usage: go test ./internal/tui/viewport.
// Related: internal/tui/viewport/viewport.go.
package viewport

import (
	"github.com/rivo/uniseg"
	"strings"
	"testing"
)

func TestScrollAnchor(t *testing.T) {
	v := View{Follow: true}
	for i := 0; i < 20; i++ {
		v.Append("abcdef", false)
	}
	v.Scroll(v.Lines(20), -8, 5)
	id := v.AnchorID
	v.Append("new", false)
	lines := v.Lines(3)
	start := v.Start(lines, 5)
	if lines[start].ID != id || v.Follow {
		t.Fatal("reading position moved")
	}
	v.Scroll(v.Lines(3), 1000, 5)
	if !v.Follow {
		t.Fatal("did not resume following")
	}
}

func TestCollapseKeepsAnchorOnItsOutput(t *testing.T) {
	v := View{Follow: true}
	v.Append("earlier conversation", false)
	v.AppendBlock(Block{Summary: "command summary", Details: strings.Repeat("detail\n", 100)})
	v.Scroll(v.Lines(80), -20, 8)
	id := v.AnchorID
	v.ToggleDetails()
	lines := v.Lines(80)
	if start := v.Start(lines, 1); lines[start].ID != id {
		t.Fatalf("folding jumped to unrelated output: %+v", lines[start])
	}
}

func TestWrapKeepsIndent(t *testing.T) {
	lines := Wrap("  abcdef\n    ghijkl", 6)
	want := []string{"  abcd", "  ef", "    gh", "    ij", "    kl"}
	if len(lines) != len(want) {
		t.Fatalf("wrapped lines: %q", lines)
	}
	for i, line := range lines {
		if line != want[i] {
			t.Fatalf("line %d=%q; want %q", i, line, want[i])
		}
	}
}

func TestWrapIndentLeavesRoomForWideCharacters(t *testing.T) {
	for _, line := range Wrap("    한글", 5) {
		if uniseg.StringWidth(line) > 5 {
			t.Fatalf("indented continuation exceeds width: %q", line)
		}
	}
}

func TestMarkdownCacheReflowsAndRefreshes(t *testing.T) {
	v := View{Follow: true}
	v.AppendBlock(Block{Text: "**한글 😀 내용**\n\n", Markdown: true})
	wide := v.Lines(80)
	narrow := v.Lines(6)
	if len(narrow) <= len(wide) {
		t.Fatal("Markdown did not reflow")
	}
	id := v.Blocks[0].ID
	v.Blocks[0].Text = "**바뀐 내용**"
	updated := v.Lines(80)
	if len(updated) != 1 || updated[0].Text != "바뀐 내용" || updated[0].ID != id {
		t.Fatalf("%+v", updated)
	}
}
