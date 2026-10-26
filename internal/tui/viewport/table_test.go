// Purpose: Verify table reflow, streamed updates and released caches retain correct visible rows and copy text.
// Pattern: Differential sparse-window presentation tests.
// Usage: go test ./internal/tui/viewport -run TestTable.
// Related: internal/tui/viewport/block_layout.go, internal/tui/markdown/table.go, internal/tui/markdown/stream.go.
package viewport

import (
	"reflect"
	"strings"
	"testing"

	"github.com/rivo/uniseg"
)

func TestTableWindowsAfterResizeAndStreaming(t *testing.T) {
	source := "before\n\n> | Name | Description | State |\n> |---|---|---:|\n> | 파일 | **" + strings.Repeat("한글 😀 abcdef ", 12) + "** | OK |\n\nafter\n\nlast"
	block := Block{ID: 7, Text: source, Markdown: true}
	for _, width := range []int{80, 34, 120, 8, 80} {
		for _, update := range []string{"", "\n\nlatest"} {
			block.Text = source + update
			want := block.Lines(width)
			if got := block.Count(width); got != len(want) {
				t.Fatalf("width %d: count %d; want %d", width, got, len(want))
			}
			for start := range want {
				if start%2 == 0 {
					block.releaseParsed()
				}
				got := block.Window(width, start, 3)
				if !reflect.DeepEqual(got, want[start:min(len(want), start+3)]) {
					t.Fatalf("width %d at %d: window differs after reflow", width, start)
				}
				for _, row := range got {
					if uniseg.StringWidth(row.Text) > width {
						t.Fatalf("overflow at width %d: %q", width, row.Text)
					}
					fragment := row.Copy
					if fragment.Row.Start < 0 || fragment.Row.End > len(fragment.Source) || fragment.Row.Start > fragment.Row.End {
						t.Fatalf("invalid selection offsets: %+v", fragment)
					}
				}
			}
		}
	}
}
