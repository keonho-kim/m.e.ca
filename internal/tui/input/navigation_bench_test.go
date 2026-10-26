// Purpose: Measure long-draft navigation and repeated edits independently of terminal I/O.
// Pattern: Editor performance regression benchmarks.
// Usage: go test ./internal/tui/input -run '^$' -bench BenchmarkLongDraft -benchmem.
// Related: internal/tui/input/editor.go, internal/tui/input/navigation.go.
package input

import (
	"strings"
	"testing"
)

func BenchmarkLongDraft(b *testing.B) {
	for _, test := range []struct{ name, text string }{
		{"ASCII", strings.Repeat("ordinary text ", 8192)},
		{"Unicode", strings.Repeat("한글 á 👩‍💻 🇰🇷 ", 8192)},
	} {
		b.Run(test.name, func(b *testing.B) {
			b.Run("MoveWindow", func(b *testing.B) {
				var editor Editor
				editor.Set(test.text)
				editor.Move(-1)
				editor.Move(1)
				editor.Window(80, 6)
				b.ReportAllocs()
				for b.Loop() {
					editor.Move(-1)
					editor.Window(80, 6)
					editor.Move(1)
					editor.Window(80, 6)
				}
			})
			b.Run("Move", func(b *testing.B) {
				var editor Editor
				editor.Set(test.text)
				editor.Move(-1)
				editor.Move(1)
				b.ReportAllocs()
				for b.Loop() {
					editor.Move(-1)
					editor.Move(1)
				}
			})
			b.Run("Edit", func(b *testing.B) {
				var editor Editor
				editor.Set(test.text)
				editor.Move(-1)
				editor.Move(1)
				b.ReportAllocs()
				for b.Loop() {
					editor.Insert("x")
					editor.Backspace()
				}
			})
		})
	}
}
