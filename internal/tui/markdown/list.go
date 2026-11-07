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
		if r.failed || r.stopped {
			return
		}
		if p := r.listProgress; p != nil && p.node == v {
			if item == v.FirstChild() {
				p.before = r.rowCount()
			}
			if item == p.mutable {
				p.row = r.rowCount()
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
		start := r.rowCount()
		continuation := indent + strings.Repeat(" ", uniseg.StringWidth(marker))
		outer := r.firstRow
		r.firstRow = func(value Line) Line {
			value = listMarker(value, continuation, indent+marker)
			if outer != nil {
				value = outer(value)
			}
			return value
		}
		r.blocks(item, continuation, depth+1)
		if r.rowCount() == start {
			r.firstRow = outer
		}
		if v.IsTight {
			r.trimBlankRows(start)
		}
	}
	r.blank()
}
