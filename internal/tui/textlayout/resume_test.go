// Purpose: Verify sparse row resumption preserves indentation, graphemes and original copy offsets.
// Pattern: Differential layout tests.
// Usage: go test ./internal/tui/textlayout -run TestRowsFrom.
// Related: internal/tui/textlayout/wrap.go, internal/tui/viewport/block_layout.go.
package textlayout

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestRowsFrom(t *testing.T) {
	for _, text := range []string{"    한글 😀 long text with spaces\n  next", strings.Repeat("abc def\t한글 ", 100), "\n\nlast\n"} {
		for _, width := range []int{1, 5, 20} {
			all := Wrap(text, width)
			for at, row := range all {
				got := slices.Collect(RowsFrom(text, width, row.Start, row.Prefix))
				if !reflect.DeepEqual(got, all[at:]) {
					t.Fatalf("width %d at %d: %+v want %+v", width, at, got, all[at:])
				}
			}
		}
	}
}
