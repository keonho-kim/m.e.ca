// Purpose: Bound chat boundary-row construction while preserving original response text.
// Pattern: Presentation behavior and allocation regression tests.
// Usage: go test ./internal/tui/viewport -run TestTrailing -bench BenchmarkTrailingRows.
// Related: internal/tui/viewport/block_layout.go, internal/tui/markdown/render.go.
package viewport

import (
	"fmt"
	"strings"
	"testing"
)

func TestTrailingChatRowsPreserveSourceAndLaterContent(t *testing.T) {
	text := "**Result**" + strings.Repeat("\n", 65536)
	for _, live := range []bool{false, true} {
		block := Block{Text: text, Markdown: true, live: live}
		if count := block.Count(80); count != 3 || len(block.layout.parsed) > 3 {
			t.Fatalf("constructed hidden rows: count=%d parsed=%d", count, len(block.layout.parsed))
		}
		if block.Text != text {
			t.Fatal("raw response was changed")
		}
		block.Text += "Following content"
		rows := block.Window(80, max(0, block.Count(80)-3), 3)
		if len(rows) == 0 || !strings.Contains(rows[len(rows)-1].Text, "Following content") {
			t.Fatal("appending after omitted boundary blanks lost content")
		}
	}
}

func BenchmarkTrailingRows(b *testing.B) {
	for _, lines := range []int{2, 65536} {
		for _, live := range []bool{false, true} {
			b.Run(fmt.Sprintf("newlines%d/live%v", lines, live), func(b *testing.B) {
				text := "A short completed response." + strings.Repeat("\n", lines)
				b.ReportAllocs()
				for b.Loop() {
					block := Block{Text: text, Markdown: true, live: live}
					count := block.Count(80)
					block.Window(80, 0, min(25, count))
				}
			})
		}
	}
}
