// Purpose: Verify display spacing ignores empty model blocks without changing literal tool content.
// Pattern: Presentation contract tests.
// Usage: go test ./internal/tui/viewport -run TestDisplay.
// Related: internal/tui/viewport/viewport.go, internal/tui/viewport/block_layout.go.
package viewport

import "testing"

func TestDisplayIgnoresWhitespaceBlocks(t *testing.T) {
	v := View{}
	for _, b := range []Block{{Text: " \n\t", Markdown: true}, {Text: "\n\n", Thinking: true}, {Text: " \n"}} {
		v.AppendBlock(b)
		if lines := b.Window(80, 0, b.Count(80)); len(lines) != 0 {
			t.Fatalf("empty block rendered: %+v", lines)
		}
	}
	if len(v.Blocks) != 0 {
		t.Fatal("whitespace blocks accumulated")
	}
}

func TestDisplayBoundsModelEdgesAndBlockGap(t *testing.T) {
	v := View{}
	raw := "\n\nReason\n\n\n\n\n"
	v.Append(raw, true)
	v.AppendBlock(Block{Text: "Answer\n\n", Markdown: true})
	v.Append("\n  write_file\n\n    content\n\n\n\n    end\n", false)
	var layout Transcript
	lines := layout.all(&v, 80)
	if lines[0].Text != "Reason" {
		t.Fatalf("leading model blank rows: %+v", lines)
	}
	if v.Blocks[0].Text != raw {
		t.Fatal("raw thinking changed")
	}
	answer, tool, content, end := -1, -1, -1, -1
	for i, line := range lines {
		switch line.Text {
		case "Answer":
			answer = i
		case "  write_file":
			tool = i
		case "    content":
			content = i
		case "    end":
			end = i
		}
	}
	if answer != 3 || tool-answer != 3 {
		t.Fatalf("unbounded display gap: %+v", lines)
	}
	if end-content != 4 {
		t.Fatalf("literal tool spacing changed: %+v", lines)
	}
}
