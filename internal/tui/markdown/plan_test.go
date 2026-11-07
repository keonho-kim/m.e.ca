// Purpose: Preserve full Markdown styling and row order across resumable block boundaries.
// Pattern: Differential full-render, random-window and fuzz tests.
// Usage: go test ./internal/tui/markdown -run TestPlan -fuzz FuzzPlan.
// Related: internal/tui/markdown/plan.go, internal/tui/markdown/code_rows.go.
package markdown

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func assertPlanRows(t *testing.T, text string, width int) {
	t.Helper()
	source := []byte(text)
	plan := newPlan(text, source, documentParser.Parse(source), width, 1<<20)
	if plan == nil {
		return
	} // Unsupported/over-budget nodes keep the existing visitor fallback.
	want := Render(text, width)
	var got []Line
	plan.Walk(0, func(value Line) bool { got = append(got, value); return true })
	if len(got) != len(want) || len(got) > 0 && !reflect.DeepEqual(got, want) {
		t.Fatalf("planned rows changed: width=%d text=%q\ngot=%#v\nwant=%#v", width, text, got, want)
	}
	for _, start := range []int{0, len(want) / 2, max(0, len(want)-1), len(want)} {
		var rows []Line
		plan.Walk(start, func(value Line) bool { rows = append(rows, value); return len(rows) < 3 })
		expected := want[start:min(len(want), start+3)]
		if len(rows) != len(expected) || len(rows) > 0 && !reflect.DeepEqual(rows, expected) {
			t.Fatal("resumed rows changed", start, text)
		}
	}
}

func TestPlanPreservesStyledRowsAndResume(t *testing.T) {
	unit := "# heading\n\nparagraph **bold** *italic*\n\n```go\n한글\ttext\n\n```\n\n- one\n  - nested\n- [x] task\n\n> quote\n\n| A | B |\n|---|---|\n| a | b |\n\n"
	for _, text := range []string{"", "\n\n", unit, strings.Repeat(unit, 20), "```\n```", "```text\n\n\n```\n\n", "    one\n    two\n\n", "[ref]: https://example.com\n\n[link][ref]\n", "- ```\n  code\n  ```\n"} {
		for _, width := range []int{1, 8, 80} {
			assertPlanRows(t, text, width)
		}
	}
}

func TestPlanCodeCheckpointsFitSmallBudget(t *testing.T) {
	var source strings.Builder
	source.WriteString("```\n")
	for index := range 10000 {
		source.WriteString(strconv.Itoa(index))
		source.WriteByte('\n')
	}
	source.WriteString("```")
	text := source.String()
	packed, plan := RenderResumable(text, 80, 64<<10)
	if packed != nil || plan == nil || plan.RetainedBytes() > 64<<10 {
		t.Fatal("source-backed plan did not fit", packed, plan)
	}
	var rows []Line
	plan.Walk(9999, func(value Line) bool { rows = append(rows, value); return false })
	if len(rows) != 1 || rows[0].Text != "│ 9999" {
		t.Fatal("checkpoint lost the final code row")
	}
}

func FuzzPlan(f *testing.F) {
	for _, text := range []string{"#", "```go\none\n\n```\n", "# title\n\nbody", "- **one**\n- two\n", "a\n\n\n", "```\n\x00\xff\n```", "> block\n", "|a|b|\n|-|-|\n|c|d|\n"} {
		f.Add(text, uint8(8))
	}
	f.Fuzz(func(t *testing.T, text string, width uint8) {
		if len(text) > 8192 {
			t.Skip()
		}
		assertPlanRows(t, text, int(width%100)+1)
	})
}
