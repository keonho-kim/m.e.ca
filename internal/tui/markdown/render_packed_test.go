// Purpose: Preserve Markdown content, styles and copying across direct compact rendering.
// Pattern: Differential renderer and bounded ownership tests.
// Usage: Markdown tests compare compact output to the established full renderer.
// Related: internal/tui/markdown/render_packed.go, internal/tui/markdown/packed_test.go.
package markdown

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestDirectPackedRenderingMatchesFullRows(t *testing.T) {
	unit := "# 제목 😀\n\n**굵게** *기울임* `code` [문서][ref]\n\n- first\n  - nested\n- [x] checked\n\n> quotation\n\n```text\n한글\tcode\n\n```\n\n---\n\n"
	for _, text := range []string{unit + "[ref]: https://example.com\n", strings.Repeat(unit, 1000) + "[ref]: https://example.com", strings.Repeat(unit, 1000) + "\n\n\n", strings.Repeat("| Name | Value |\n|---|---|\n| **파일** | value 한글 |\n\n", 100)} {
		for _, width := range []int{8, 30, 80} {
			want := Render(text, width)
			got := RenderPacked(text, width, 16<<20)
			if got == nil || got.Len() != len(want) || got.RetainedBytes() > 16<<20 {
				t.Fatalf("width=%d: compact row count or budget changed", width)
			}
			for i, line := range want {
				if got.Text(i) != line.Text || !reflect.DeepEqual(slices.Collect(got.RowsFrom(i, width, 0, "")), slices.Collect(line.RowsFrom(width, 0, ""))) {
					t.Fatalf("width=%d row=%d: content, styles or copy source changed", width, i)
				}
			}
		}
	}
	if RenderPacked(strings.Repeat(unit, 100), 80, 128) != nil {
		t.Fatal("over-budget compact result retained")
	}
}
