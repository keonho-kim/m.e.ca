// Purpose: Verify centered rows reflow with terminal width and preserve exact source copying.
// Pattern: Pure layout contract tests.
// Usage: go test ./internal/tui/viewport -run TestCentered.
// Related: internal/tui/viewport/alignment.go, internal/tui/viewport/viewport.go.
package viewport

import (
	"strings"
	"testing"

	"github.com/rivo/uniseg"
)

func TestCenteredRowsReflowAndCopy(t *testing.T) {
	source := "Worked for 01:02:03 | INPUT 120 | OUTPUT 20 | TPS 5.0"
	v := View{Follow: true}
	v.AppendBlock(Block{Text: source, Centered: true, Thinking: true})
	for _, width := range []int{100, 18, 80} {
		var copied strings.Builder
		for _, line := range v.Lines(width) {
			text := strings.TrimLeft(line.Text, " ")
			padding := (width - uniseg.StringWidth(text)) / 2
			if line.Text != strings.Repeat(" ", padding)+text || !line.Thinking {
				t.Fatalf("width %d: %+v", width, line)
			}
			copy := line.Copy
			start, end := copy.Row.Offset(copy.Source, 0, false), copy.Row.Offset(copy.Source, width-1, true)
			copied.WriteString(copy.Source[start:end])
		}
		if copied.String() != source {
			t.Fatalf("width %d: copy=%q", width, copied.String())
		}
	}
	v.Blocks[0].Centered = false
	if v.Lines(100)[0].Text != source {
		t.Fatal("alignment change reused stale layout")
	}
}

func TestCenteredRowsUseUnicodeCellWidth(t *testing.T) {
	block := Block{Text: "작업 시간 00:01:02", Centered: true}
	line := block.Lines(80)[0]
	padding := (80 - uniseg.StringWidth(block.Text)) / 2
	if line.Text != strings.Repeat(" ", padding)+block.Text {
		t.Fatal(line.Text)
	}
	if got := line.Copy.Row.Offset(block.Text, padding, true); got != len("작") {
		t.Fatalf("center padding corrupted hit testing: %d", got)
	}
}
