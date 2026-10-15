// Purpose: Project Markdown tables with bounded cell padding and visible header emphasis.
// Pattern: Pure table layout.
// Usage: The block renderer calls table for GFM table nodes.
// Related: internal/tui/markdown/render.go, internal/tui/markdown/inline.go.
package markdown

import (
	"github.com/rivo/uniseg"
	"github.com/yuin/goldmark/v2/ast"
	extast "github.com/yuin/goldmark/v2/extension/ast"
	"strings"
)

func (r *renderer) table(table *extast.Table, indent string) {
	var rows [][][]Span
	var headers []bool
	var collect func(ast.Node)
	collect = func(parent ast.Node) {
		for node := parent.FirstChild(); node != nil; node = node.NextSibling() {
			if _, ok := node.(*extast.TableBody); ok {
				collect(node)
				continue
			}
			_, header := node.(*extast.TableHeader)
			var cells [][]Span
			for cell := node.FirstChild(); cell != nil; cell = cell.NextSibling() {
				style := Style(0)
				if header {
					style = Bold
				}
				cells = append(cells, r.inline(cell, style, 0))
			}
			rows = append(rows, cells)
			headers = append(headers, header)
		}
	}
	collect(table)
	var widths []int
	for _, row := range rows {
		for i, cell := range row {
			if i >= len(widths) {
				widths = append(widths, 0)
			}
			widths[i] = max(widths[i], min(32, uniseg.StringWidth(line(cell).Text)))
		}
	}
	for index, row := range rows {
		var spans []Span
		for i, cell := range row {
			if i > 0 {
				spans = add(spans, " │ ", Dim)
			}
			spans = append(spans, cell...)
			spans = add(spans, strings.Repeat(" ", max(0, widths[i]-uniseg.StringWidth(line(cell).Text))), 0)
		}
		r.emit(indent, spans)
		if headers[index] {
			var separators []string
			for _, width := range widths {
				separators = append(separators, strings.Repeat("─", max(1, width)))
			}
			r.emit(indent, []Span{{Text: strings.Join(separators, "─┼─"), Style: Dim}})
		}
	}
}
