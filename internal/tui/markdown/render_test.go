// Purpose: Verify Markdown styling, literal code and Unicode-aware terminal rows.
// Pattern: Presentation contract tests.
// Usage: go test ./internal/tui/markdown.
// Related: internal/tui/markdown/render.go, internal/tui/markdown/layout.go.
package markdown

import (
	"github.com/rivo/uniseg"
	"slices"
	"strings"
	"testing"
)

func TestMarkdownRendering(t *testing.T) {
	input := "# 제목 😀\n\n**굵게** *강조* 및 `x_y`\n\n- 항목\n\n> 인용\n\n```go\n  a := \"**literal**\"\n```\n\n[문서](https://example.com)\n"
	rows := Render(input, 80)
	var text strings.Builder
	bold, italic, code := false, false, false
	for _, row := range rows {
		for _, span := range row.Spans {
			text.WriteString(span.Text)
			bold = bold || span.Style&Bold != 0
			italic = italic || span.Style&Italic != 0
			code = code || span.Text == "x_y" && span.Style&Underline != 0
		}
		text.WriteByte('\n')
	}
	for _, want := range []string{"제목 😀", "굵게", "강조", "항목", "인용", `  a := "**literal**"`, "https://example.com"} {
		if !strings.Contains(text.String(), want) {
			t.Fatalf("missing %s: %s", want, text.String())
		}
	}
	if strings.Contains(text.String(), "# 제목") || !bold || !italic || !code {
		t.Fatalf("styles not rendered: %s", text.String())
	}
}
func TestWrapKeepsGraphemesAndStyles(t *testing.T) {
	source := []Span{{Text: "한글👨‍👩‍👧‍👦 **", Style: Bold}, {Text: "끝", Style: Italic}}
	rows := slices.Collect(Rows(source, 6))
	var joined strings.Builder
	for _, row := range rows {
		if uniseg.StringWidth(row.Text) > 6 {
			t.Fatalf("overflow %q", row.Text)
		}
		for _, span := range row.Spans {
			joined.WriteString(span.Text)
		}
	}
	if joined.String() != "한글👨‍👩‍👧‍👦 **끝" {
		t.Fatalf("graphemes lost: %q", joined.String())
	}
}
func TestTableAndUnfinishedFence(t *testing.T) {
	for _, input := range []string{"| 이름 | 값 |\n|---|---|\n| A | 2 |", "~~~\n**raw**"} {
		rows := Render(input, 80)
		var text strings.Builder
		for _, row := range rows {
			text.WriteString(row.Text)
		}
		if input[0] == '|' && !strings.Contains(text.String(), "A") {
			t.Fatal(text.String())
		}
		if input[0] == '~' && !strings.Contains(text.String(), "**raw**") {
			t.Fatal(text.String())
		}
	}
}

func TestTildesRemainLiteral(t *testing.T) {
	input := "3~5일, 10~20개, ~문장~ 그리고 ~~문장~~"
	var body strings.Builder
	for _, row := range Render(input, 80) {
		body.WriteString(row.Text)
	}
	if body.String() != input {
		t.Fatalf("tilde interpreted as markup: %q", body.String())
	}
}

func TestDecodedControlsStaySafeAndEmojiStayWhole(t *testing.T) {
	rows := Render("&#27;[31m **👨‍👩‍👧‍👦** ~3~5~", 80)
	var body strings.Builder
	for _, row := range rows {
		body.WriteString(row.Text)
	}
	if strings.ContainsRune(body.String(), '\x1b') || !strings.Contains(body.String(), "👨‍👩‍👧‍👦") || !strings.Contains(body.String(), "~3~5~") {
		t.Fatal(body.String())
	}
}
