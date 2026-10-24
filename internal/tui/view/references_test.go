// Purpose: Verify selected path displays preserve Unicode and fit their allocated cells and rows.
// Pattern: Pure presentation contract tests.
// Usage: go test ./internal/tui/view -run TestReferenceLines.
// Related: internal/tui/view/references.go, internal/tui/view/composer.go.
package view

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/keonho-kim/meca/internal/tui/locale"
	"github.com/rivo/uniseg"
)

func TestReferenceLinesFitWithoutSplittingUnicode(t *testing.T) {
	paths := []string{strings.Repeat("디렉터리/", 16) + "👨‍👩‍👧‍👦.md", "src/server.py"}
	for _, width := range []int{8, 28, 80} {
		for _, language := range []locale.Language{locale.Korean, locale.English} {
			lines := ReferenceLines(paths, width, 1, language, func(s string) string { return s })
			if len(lines) != 1 || !strings.HasSuffix(lines[0], "…") || !utf8.ValidString(lines[0]) || uniseg.StringWidth(lines[0]) > width {
				t.Fatalf("width=%d language=%v: %q", width, language, lines)
			}
		}
	}
	if ReferenceLines(nil, 80, 3, locale.English, func(s string) string { return s }) != nil || ReferenceLines(paths, 80, 0, locale.English, func(s string) string { return s }) != nil {
		t.Fatal("empty or unallocated references consumed display rows")
	}
}

func BenchmarkReferenceLines(b *testing.B) {
	for _, count := range []int{1000, 10000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			paths := make([]string, count)
			for i := range paths {
				paths[i] = fmt.Sprintf("src/package_%05d/implementation.go", i)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				ReferenceLines(paths, 80, 3, locale.English, func(s string) string { return s })
			}
		})
	}
}
