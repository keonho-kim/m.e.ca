// Purpose: Render list items and expose safe item boundaries for incremental streams.
// Pattern: AST-to-terminal adapter with one optional checkpoint.
// Usage: renderer.block delegates lists; Stream observes the last two mutable items.
// Related: internal/tui/markdown/render.go, internal/tui/markdown/stream_list.go.
package markdown

import (
	"fmt"
	"strings"

	"github.com/rivo/uniseg"
	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/extension"
)

func (r *renderer) list(v *ast.List, indent string, depth int) {
	index := v.Start
	for item := v.FirstChild(); item != nil; item = item.NextSibling() {
		if p := r.listProgress; p != nil && p.node == v {
			if item == v.FirstChild() {
				p.before = len(r.rows)
			}
			if item == p.mutable {
				p.row = len(r.rows)
			}
		}
		marker := "• "
		if v.IsOrdered() {
			marker = fmt.Sprintf("%d. ", index)
			index++
		}
		if status, ok := extension.TaskStatusOf(item); ok {
			marker = "[ ] "
			if status == extension.TaskStatusCompleted {
				marker = "[x] "
			}
		}
		start := len(r.rows)
		continuation := indent + strings.Repeat(" ", uniseg.StringWidth(marker))
		r.blocks(item, continuation, depth+1)
		if len(r.rows) > start {
			spans := r.rows[start].Spans
			if len(spans) > 0 && strings.HasPrefix(spans[0].Text, continuation) {
				spans[0].Text = strings.TrimPrefix(spans[0].Text, continuation)
			}
			r.rows[start] = line(append([]Span{{Text: indent + marker, Style: Dim}}, spans...))
		}
		if v.IsTight {
			for len(r.rows) > start && r.rows[len(r.rows)-1].Text == "" {
				r.rows = r.rows[:len(r.rows)-1]
			}
		}
	}
	r.blank()
}
