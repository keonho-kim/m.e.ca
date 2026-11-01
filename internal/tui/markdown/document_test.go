// Purpose: Preserve Markdown boundaries, styles and references across independent rendering groups.
// Pattern: Serial-reference regression test.
// Usage: Run TestDocumentGroups with -race.
// Related: internal/tui/markdown/document.go, internal/tui/markdown/render.go.
package markdown

import (
	"reflect"
	"strings"
	"testing"
)

func TestDocumentGroups(t *testing.T) {
	section := "# 제목\n\nFirst **bold** [reference][ref].\n\n<!-- literal -->\n\n> 한글 👨‍👩‍👧‍👦\n\n- item\n  - nested\n\n```go\ncode\n\n```\n\n| key | value |\n|---|---|\n| a | Long value |\n\n---\n\n"
	plain := strings.ReplaceAll(section, "| key | value |\n|---|---|\n| a | Long value |\n\n", "")
	for _, fixture := range []string{plain, section} {
		text := strings.Repeat(fixture, 1000) + "[ref]: https://example.com\n"
		for _, width := range []int{8, 80} {
			source := []byte(text)
			doc := documentParser.Parse(source)
			serial := renderer{source: source, width: width}
			serial.blocks(doc, "", 0)
			parallel := renderer{source: source, width: width}
			parallel.document(doc)
			if serial.failed != parallel.failed || !reflect.DeepEqual(serial.rows, parallel.rows) {
				t.Fatalf("grouped rendering changed rows at width %d", width)
			}
		}
	}
}
