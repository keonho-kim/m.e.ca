// Purpose: Preserve rendered text, styling, Unicode and copy coordinates through compact row storage.
// Pattern: Differential presentation and resource-bound tests.
// Usage: go test ./internal/tui/markdown -run TestPacked.
// Related: internal/tui/markdown/packed.go, internal/tui/markdown/layout.go.
package markdown

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestPackedRowsMatchRenderedWindows(t *testing.T) {
	text := "# 제목 😀\n\n**굵게** *기울임* `code` [문서][ref]\n\n- first\n  - nested\n- [x] checked\n\n> quotation\n\n```text\n한글\tcode\n```\n\n| Name | Value |\n|---|---:|\n| **파일** | long value 한글 😀 |\n\n[ref]: https://example.com/한글\n"
	for _, width := range []int{1, 8, 30, 80} {
		lines := Render(text, width)
		packed := Pack(lines, 1<<20)
		if packed == nil || packed.Len() != len(lines) || packed.RetainedBytes() > 1<<20 {
			t.Fatal("rendered rows were not packed within budget")
		}
		for i, line := range lines {
			for _, start := range []int{0, len(line.Text) / 2, len(line.Text)} {
				// Resume only at a valid rendered byte boundary.
				for start > 0 && start < len(line.Text) && line.Text[start]&0xc0 == 0x80 {
					start--
				}
				want := slices.Collect(line.RowsFrom(width, start, ""))
				got := slices.Collect(packed.RowsFrom(i, width, start, ""))
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("width=%d line=%d start=%d: packed rows differ", width, i, start)
				}
			}
		}
	}
}

func TestPackedManyStylesResumeWithoutMaterializingAllSpans(t *testing.T) {
	var spans []Span
	for i := range 10000 {
		spans = append(spans, Span{Text: "abc", Style: Style(i % 16)})
	}
	logical := line(spans)
	packed := Pack([]Line{logical}, 1<<20)
	if packed == nil {
		t.Fatal("styled row did not fit")
	}
	want := slices.Collect(logical.RowsFrom(40, len(logical.Text)-80, "  "))
	got := slices.Collect(packed.RowsFrom(0, 40, len(logical.Text)-80, "  "))
	if !reflect.DeepEqual(got, want) {
		t.Fatal("style boundaries or copied source changed")
	}
	allocs := testing.AllocsPerRun(10, func() {
		for range packed.RowsFrom(0, 40, len(logical.Text)-80, "  ") {
		}
	})
	if allocs > 100 {
		t.Fatalf("visible rows allocated all %d styles: %g allocations", len(spans), allocs)
	}
}

func TestPackedBudgetRejectsCacheWithoutChangingRows(t *testing.T) {
	lines := Render(strings.Repeat("**value**\n\n", 100), 80)
	if Pack(lines, 128) != nil {
		t.Fatal("oversized cache retained")
	}
	if lines[0].Text != "value" {
		t.Fatal("cache admission changed original rows")
	}
	if Pack([]Line{{Text: "unbound", Spans: []Span{{Text: "different"}}}}, 4096) != nil {
		t.Fatal("inconsistent text/style ranges admitted")
	}
}

func TestPackedOwnsTextAndDoesNotExposeStyleArrays(t *testing.T) {
	input := Render("**original** and `value`", 80)
	want := slices.Collect(input[0].RowsFrom(80, 0, ""))
	packed := Pack(input, 4096)
	if packed == nil {
		t.Fatal("small snapshot rejected")
	}
	input[0].Text = "changed"
	input[0].Spans[0].Text = "changed"
	got := slices.Collect(packed.RowsFrom(0, 80, 0, ""))
	if !reflect.DeepEqual(got, want) {
		t.Fatal("input mutation changed packed text")
	}
	got[0].Spans[0].Style = 0
	if replay := slices.Collect(packed.RowsFrom(0, 80, 0, "")); !reflect.DeepEqual(replay, want) {
		t.Fatal("consumer mutation changed cached styles")
	}
}
