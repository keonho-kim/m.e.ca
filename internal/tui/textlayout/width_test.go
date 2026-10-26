// Purpose: Verify the fast column count preserves Unicode and control semantics.
// Pattern: Differential layout test and benchmark.
// Usage: go test ./internal/tui/textlayout -run TestWidth -bench BenchmarkWidth.
// Related: internal/tui/textlayout/width.go.
package textlayout

import (
	"strings"
	"testing"

	"github.com/rivo/uniseg"
)

func TestWidthMatchesUnicodeReference(t *testing.T) {
	for _, text := range []string{"", "ASCII text 0123456789", "한글", "a\u0301", "👨‍👩‍👧‍👦 🇰🇷", "\n\t\r\x00\x7f", "some ASCII before 한글", "\xff"} {
		if got, want := Width(text), uniseg.StringWidth(text); got != want {
			t.Fatalf("%q: width=%d; want %d", text, got, want)
		}
	}
}

func BenchmarkWidth(b *testing.B) {
	for _, text := range []string{strings.Repeat("ASCII text ", 8), strings.Repeat("한글 👩‍💻 a\u0301 ", 8)} {
		for name, measure := range map[string]func(string) int{"reference": uniseg.StringWidth, "fast": Width} {
			b.Run(name+"/"+text[:5], func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if measure(text) < 1 {
						b.Fatal("empty width")
					}
				}
			})
		}
	}
}
