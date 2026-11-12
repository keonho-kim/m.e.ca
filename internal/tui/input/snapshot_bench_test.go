// Purpose: Measure large-draft mutation and visible-window allocation with warm grapheme indexes.
// Pattern: Editor allocation regression benchmark.
// Usage: go test ./internal/tui/input -run '^$' -bench BenchmarkDraftSnapshots -benchmem.
// Related: internal/tui/input/editor.go, internal/tui/input/layout.go.
package input

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkDraftSnapshots(b *testing.B) {
	for _, size := range []int{1 << 20, 8 << 20} {
		b.Run(fmt.Sprintf("%dMiB", size>>20), func(b *testing.B) {
			var editor Editor
			editor.Set(strings.Repeat("a", size))
			editor.Move(-1)
			editor.Move(1)
			editor.Window(80, 6)
			b.ReportAllocs()
			for b.Loop() {
				editor.Insert("x")
				editor.Window(80, 6)
				editor.Backspace()
				editor.Window(80, 6)
			}
		})
	}
}
