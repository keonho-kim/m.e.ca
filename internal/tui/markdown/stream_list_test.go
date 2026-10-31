// Purpose: Preserve full Markdown semantics when completed list items are reused.
// Pattern: Differential presentation and resource tests.
// Usage: Run TestStreamList and FuzzStream after changing list checkpoints.
// Related: internal/tui/markdown/stream_list.go, internal/tui/markdown/stream.go.
package markdown

import (
	"strings"
	"testing"
)

func TestStreamListMatchesWholeDocument(t *testing.T) {
	for _, text := range []string{
		strings.Repeat("- **item** and `code`\n", 10) + "\n- loose\n\n  continuation\n\nend\n",
		strings.Repeat("1. numbered\n", 10) + "2. last\n\n# heading\n\nend",
		"# first\n\nparagraph\n\n" + strings.Repeat("- item\n  - nested\n  - child\n", 8) + "\n- next\n",
		strings.Repeat("- item\n\n", 8) + "- item\n- tight looking\n- last\n",
		strings.Repeat("- [x] done\n- [ ] todo\n", 5) + "\n---\n\nother",
		strings.Repeat("- item\n", 5) + "\n    code\n\n\tcode\n- last\n",
	} {
		var stream Stream
		for i := 0; i <= len(text); i++ {
			if got, want := stream.Render(text[:i], 80), Render(text[:i], 80); !equalLines(got, want) {
				t.Fatalf("prefix %q:\ngot %#v\nwant %#v", text[:i], got, want)
			}
		}
		if stream.RetainedBytes() == 0 {
			t.Fatal("retained resources were not accounted")
		}
	}
}

func TestStreamListReusesCompletedItems(t *testing.T) {
	text := strings.Repeat("1. **result** and `code`\n", 4000)
	var stream Stream
	stream.Render(text, 80)
	got := stream.Render(text+"1. next", 80)
	if !equalLines(got, Render(text+"1. next", 80)) || stream.ReusedRows() < 3990 {
		t.Fatal("list prefix was rebuilt or numbering changed", stream.ReusedRows())
	}
	if !equalLines(stream.Render("reset", 80), Render("reset", 80)) {
		t.Fatal("replacement retained a stale list")
	}
}
