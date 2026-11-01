// Purpose: Preserve complete Markdown semantics while open code blocks reuse finished lines.
// Pattern: Differential streaming regression.
// Usage: Run TestStreamFence and FuzzStream after checkpoint changes.
// Related: internal/tui/markdown/stream_fence.go, internal/tui/markdown/stream.go.
package markdown

import (
	"strings"
	"testing"
)

func TestStreamFenceMatchesFullRendering(t *testing.T) {
	for _, source := range []string{
		"```go\r\nx := 1\r\n\r\n```\r\n\r\nafter",
		"```go\nx := 1\n\nvalue\n```\n\nafter",
		"before\n\nparagraph\n\n~~~lang\na\n\n\n끝 😀\n~~~\n\n# heading",
		"```\n\n\n",
		"````lang\n```\nnot closed\n  ````\n\nafter",
		"```\nref ]: literal\n~~~\n```\n\n[a][b]\n\n[b]: https://example.com",
	} {
		var stream Stream
		for i := 0; i <= len(source); i++ {
			if got, want := stream.Render(source[:i], 80), Render(source[:i], 80); !equalLines(got, want) {
				t.Fatalf("prefix %q:\ngot %#v\nwant %#v", source[:i], got, want)
			}
		}
	}
}

func TestStreamFenceReusesLargeBody(t *testing.T) {
	text := "```go\n" + strings.Repeat("value := 1\n", 4000)
	var stream Stream
	stream.Render(text, 80)
	got := stream.Render(text+"next", 80)
	if !equalLines(got, Render(text+"next", 80)) || stream.ReusedRows() < 3990 {
		t.Fatalf("code prefix rebuilt: %d", stream.ReusedRows())
	}
	if !equalLines(stream.Render("reset", 80), Render("reset", 80)) {
		t.Fatal("replacement retained stale code")
	}
}
