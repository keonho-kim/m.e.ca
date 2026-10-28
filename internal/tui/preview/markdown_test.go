// Purpose: Verify styled preview bounds, raw switching and content-cache invalidation.
// Pattern: Presentation contract tests.
// Usage: go test ./internal/tui/preview -run TestMarkdownLayout.
// Related: internal/tui/preview/markdown.go, internal/tui/preview/raw.go.
package preview

import (
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/tui/markdown"
	"github.com/rivo/uniseg"
)

func TestMarkdownLayoutModesAndRefresh(t *testing.T) {
	p := Layout{Path: "guide.markdown", Content: "# Heading\n\n**bold** text\n\n| Name | Detail |\n| --- | --- |\n| item | a longer cell body |\n\n```mermaid\ngraph TD\nA --> B\n```\n"}
	for _, width := range []int{80, 26} {
		rows := p.Window(width, 0, p.Count(width))
		var text strings.Builder
		bold := false
		for _, row := range rows {
			if uniseg.StringWidth(row.Text) > width {
				t.Fatalf("preview exceeded %d columns: %q", width, row.Text)
			}
			text.WriteString(row.Text + "\n")
			for _, span := range row.Spans {
				bold = bold || span.Text == "Heading" && span.Style&markdown.Bold != 0
			}
		}
		if !p.Rendered() || !bold || strings.Contains(text.String(), "# Heading") || strings.Contains(text.String(), "**bold**") || !strings.Contains(text.String(), "A --> B") {
			t.Fatalf("rendered Markdown or Mermaid source lost: %s", text.String())
		}
	}
	p = Layout{Path: p.Path, Content: p.Content, Raw: true}
	rows := p.Window(80, 0, 3)
	if p.Rendered() || len(rows) == 0 || rows[0].Text != "  1 | # Heading" || len(rows[0].Spans) != 0 {
		t.Fatalf("raw preview lost numbering or changed the source: %+v", rows)
	}
	if p.formatted.Text != "" {
		t.Fatal("raw preview retained its inactive Markdown cache")
	}
	p = Layout{Path: p.Path, Content: p.Content}
	p.Content, p.Truncated = "# Updated\n", true
	rows = p.Window(80, 0, p.Count(80))
	var text strings.Builder
	for _, row := range rows {
		text.WriteString(row.Text + "\n")
	}
	if strings.Contains(text.String(), "Heading") || !strings.Contains(text.String(), "Updated") || rows[len(rows)-1].Text != previewTruncated {
		t.Fatalf("changed content reused old rows or lost truncation: %s", text.String())
	}
	p.Content = ""
	p.Count(80)
	if p.formatted.Text != "" || p.content != "" || p.index != nil {
		t.Fatal("empty preview retained its previous layout or source")
	}
}

func BenchmarkMarkdownLayoutWindow(b *testing.B) {
	p := Layout{Path: "guide.md", Content: strings.Repeat("## Heading\n\nA **formatted** paragraph.\n\n", 4096)}
	count := p.Count(80)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		p.Window(80, count-24, 24)
	}
}
