// Purpose: Verify shared parser configuration never shares document references or styled output.
// Pattern: Concurrent presentation contract tests.
// Usage: go test -race ./internal/tui/markdown -run TestParser.
// Related: internal/tui/markdown/parser.go, internal/tui/markdown/render.go.
package markdown

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestParserDocumentIsolation(t *testing.T) {
	var workers sync.WaitGroup
	for i := range 12 {
		workers.Go(func() {
			url := fmt.Sprintf("https://example.com/document/%d", i)
			text := "[label][id]\n\n[id]: " + url + "\n\n- [x] done\n\n| A | B |\n|---|---|\n| 한글 | 😀 |"
			for range 12 {
				var result strings.Builder
				for _, row := range Render(text, 40) {
					result.WriteString(row.Text)
				}
				if !strings.Contains(result.String(), url) || !strings.Contains(result.String(), "[x] done") {
					t.Error("another document affected rendering", result.String())
				}
				var stream Stream
				for _, source := range []string{text, text + "\n\nnext", "[label][id]"} {
					if !equalLines(stream.Render(source, 40), Render(source, 40)) {
						t.Error("streamed parse retained a previous document")
					}
				}
			}
		})
	}
	workers.Wait()
}

func TestParagraphAssemblyPreservesStylesAndBreaks(t *testing.T) {
	text := strings.Repeat("plain\n", 512) + "**bold _nested_ end**  \nnext `code` &amp;\nlast"
	rows := Render(text, 80)
	if len(rows) != 2 || !strings.HasPrefix(rows[0].Text, strings.Repeat("plain ", 512)) || !strings.HasSuffix(rows[0].Text, "bold nested end") || rows[1].Text != "next code & last" {
		t.Fatal("paragraph assembly changed text or hard breaks")
	}
	styles := map[string]Style{}
	for _, row := range rows {
		for _, span := range row.Spans {
			styles[span.Text] = span.Style
		}
	}
	if styles["nested"] != Bold|Italic || styles["bold "] != Bold || styles[" end"] != Bold || styles["code"] != Underline {
		t.Fatal("paragraph assembly changed nested styles", styles)
	}
}
