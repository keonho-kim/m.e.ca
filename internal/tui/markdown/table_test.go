// Purpose: Verify long table cells stay inside aligned columns without losing styled Unicode content.
// Pattern: Terminal presentation contract tests.
// Usage: go test ./internal/tui/markdown -run TestTable.
// Related: internal/tui/markdown/table.go, internal/tui/markdown/layout.go.
package markdown

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/rivo/uniseg"
)

func TestTableWrapsCellsWithinColumnBoundaries(t *testing.T) {
	content := "한글과 👨‍👩‍👧‍👦 이모지를 포함하는 긴 설명 " + strings.Repeat("abcdefgh", 12)
	source := "| Key | Description | State |\n|:---|:---|---:|\n| item | **" + content + "** | DONE |"
	for _, width := range []int{40, 64, 120} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			rows := displayedTable(source, width)
			boundaries := tableBoundaries(rows[0].Text)
			if len(boundaries) != 2 {
				t.Fatalf("missing header columns: %q", rows[0].Text)
			}
			var columns [3]strings.Builder
			body, styled := false, false
			for _, row := range rows {
				if uniseg.StringWidth(row.Text) > width {
					t.Fatalf("row exceeds %d cells: %q", width, row.Text)
				}
				if strings.ContainsRune(row.Text, '┼') {
					body = true
					continue
				}
				if !slices.Equal(boundaries, tableBoundaries(row.Text)) {
					t.Fatalf("columns moved from %v: %q", boundaries, row.Text)
				}
				if body {
					for i, part := range strings.Split(row.Text, " │ ") {
						columns[i].WriteString(part)
					}
					for _, span := range row.Spans {
						styled = styled || strings.Contains(span.Text, "abc") && span.Style&Bold != 0
					}
				}
			}
			for i, want := range []string{"item", content, "DONE"} {
				if got := compactTableText(columns[i].String()); got != compactTableText(want) {
					t.Fatalf("column %d lost content: %q; want %q", i, got, want)
				}
			}
			if !styled {
				t.Fatal("wrapped cells lost emphasis")
			}
		})
	}
}

func TestTableHonorsColumnAlignment(t *testing.T) {
	rows := displayedTable("| Left | Right | Center |\n|:---|---:|:---:|\n| a | 5 | b |", 80)
	want := []string{"a   ", "    5", "  b   "}
	if got := strings.Split(rows[len(rows)-1].Text, " │ "); !slices.Equal(got, want) {
		t.Fatalf("alignment = %q; want %q", got, want)
	}
}

func TestTableNarrowScreenPreservesFields(t *testing.T) {
	source := "| Name | Description | State |\n|---|---|---|\n| 파일 | 한글👨‍👩‍👧‍👦abcdef | READY |"
	for _, width := range []int{2, 8, 16} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			var body strings.Builder
			for _, row := range displayedTable(source, width) {
				if uniseg.StringWidth(row.Text) > width || strings.ContainsRune(row.Text, '│') {
					t.Fatalf("narrow table did not become readable fields: %q", row.Text)
				}
				body.WriteString(row.Text)
			}
			for _, want := range []string{"Name", "Description", "State", "파일", "한글👨‍👩‍👧‍👦abcdef", "READY"} {
				if !strings.Contains(compactTableText(body.String()), want) {
					t.Fatalf("missing %q at width %d: %q", want, width, body.String())
				}
			}
		})
	}
}

func TestTableWrapsHeadersAndKeepsEmptyCells(t *testing.T) {
	source := "| Long header with 한글 and 😀 | Empty | Code |\n|---|---|---|\n| | | `abcdefghijklmnopqrstuvwxyz` |\n| value | | |"
	rows := displayedTable(source, 36)
	boundaries := tableBoundaries(rows[0].Text)
	var code strings.Builder
	for _, row := range rows {
		if strings.ContainsRune(row.Text, '┼') {
			continue
		}
		if !slices.Equal(boundaries, tableBoundaries(row.Text)) {
			t.Fatalf("header, empty or wrapped cell moved a column: %q", row.Text)
		}
		for _, span := range row.Spans {
			if span.Style&Underline != 0 {
				code.WriteString(span.Text)
			}
		}
	}
	if code.String() != "abcdefghijklmnopqrstuvwxyz" {
		t.Fatalf("wrapped inline code lost content or style: %q", code.String())
	}
}

func TestTableFitsOrderedListIndentation(t *testing.T) {
	source := "10. | Name | Description |\n    |---|---|\n    | 파일 | " + strings.Repeat("설명 text ", 12) + " |"
	rows := displayedTable(source, 40)
	boundaries := tableBoundaries(rows[0].Text)
	if len(boundaries) != 1 {
		t.Fatalf("ordered-list header broke outside the table: %q", rows[0].Text)
	}
	for _, row := range rows {
		if row.Text == "" || strings.ContainsRune(row.Text, '┼') {
			continue
		}
		if !slices.Equal(boundaries, tableBoundaries(row.Text)) {
			t.Fatalf("list marker changed the column boundary: %q", row.Text)
		}
	}
}

func displayedTable(source string, width int) []Line {
	var out []Line
	for _, row := range Render(source, width) {
		out = append(out, slices.Collect(Rows(row.Spans, width))...)
	}
	return out
}

func tableBoundaries(text string) []int {
	var positions []int
	column := 0
	for graphemes := uniseg.NewGraphemes(text); graphemes.Next(); {
		if graphemes.Str() == "│" {
			positions = append(positions, column)
		}
		column += graphemes.Width()
	}
	return positions
}

func compactTableText(text string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, text)
}
