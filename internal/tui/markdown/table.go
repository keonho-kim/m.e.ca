// Purpose: Render Markdown table cells within shared column boundaries and retain inline styling.
// Pattern: Pure table layout.
// Usage: The block renderer calls table for GFM table nodes.
// Related: internal/tui/markdown/render.go, internal/tui/markdown/table_widths.go, internal/tui/markdown/inline.go.
package markdown

import (
	"slices"
	"strings"

	"github.com/rivo/uniseg"
	"github.com/yuin/goldmark/v2/ast"
	extast "github.com/yuin/goldmark/v2/extension/ast"
)

type tableCell struct {
	spans []Span
	align extast.Alignment
}

func (r *renderer) table(table *extast.Table, indent string) {
	var rows [][]tableCell
	var headers []bool
	var collect func(ast.Node)
	collect = func(parent ast.Node) {
		for node := parent.FirstChild(); node != nil; node = node.NextSibling() {
			if _, ok := node.(*extast.TableBody); ok {
				collect(node)
				continue
			}
			_, header := node.(*extast.TableHeader)
			var cells []tableCell
			for cell := node.FirstChild(); cell != nil; cell = cell.NextSibling() {
				style := Style(0)
				if header {
					style = Bold
				}
				spans := r.inline(cell, style, 0)
				for i := range spans {
					spans[i].Text = strings.ReplaceAll(spans[i].Text, "\t", "    ")
				}
				cells = append(cells, tableCell{spans: spans, align: cell.(*extast.TableCell).Alignment})
			}
			rows = append(rows, cells)
			headers = append(headers, header)
		}
	}
	collect(table)
	available := max(1, r.width-uniseg.StringWidth(indent))
	widths := tableWidths(rows, available)
	if len(widths) == 0 {
		r.tableFields(rows, indent, available)
		return
	}
	for index, row := range rows {
		cells := make([][]Line, len(widths))
		height := 1
		for i, cell := range row {
			cells[i] = slices.Collect(Rows(cell.spans, widths[i]))
			height = max(height, len(cells[i]))
		}
		for y := 0; y < height; y++ {
			var spans []Span
			for i, width := range widths {
				if i > 0 {
					spans = add(spans, " │ ", Dim)
				}
				var value Line
				if y < len(cells[i]) {
					value = cells[i][y]
				}
				padding := max(0, width-uniseg.StringWidth(value.Text))
				left := 0
				if i < len(row) {
					switch row[i].align {
					case extast.AlignRight:
						left = padding
					case extast.AlignCenter:
						left = padding / 2
					}
				}
				spans = add(spans, strings.Repeat(" ", left), 0)
				spans = append(spans, value.Spans...)
				spans = add(spans, strings.Repeat(" ", padding-left), 0)
			}
			r.emit(indent, spans)
		}
		if headers[index] {
			var separators []string
			for _, width := range widths {
				separators = append(separators, strings.Repeat("─", max(1, width)))
			}
			r.emit(indent, []Span{{Text: strings.Join(separators, "─┼─"), Style: Dim}})
		}
	}
}

// tableFields preserves every value when readable columns cannot fit side by side.
func (r *renderer) tableFields(rows [][]tableCell, indent string, width int) {
	if len(rows) == 0 {
		return
	}
	if len(rows) == 1 {
		for _, cell := range rows[0] {
			for row := range Rows(cell.spans, width) {
				r.emit(indent, row.Spans)
			}
		}
		return
	}
	padding := strings.Repeat(" ", min(2, max(0, width-2)))
	for index, cells := range rows[1:] {
		if index > 0 {
			r.blank()
		}
		for i, cell := range cells {
			if i < len(rows[0]) {
				label := slices.Clone(rows[0][i].spans)
				label = add(label, ":", Bold)
				for row := range Rows(label, width) {
					r.emit(indent, row.Spans)
				}
			}
			for row := range Rows(cell.spans, width-len(padding)) {
				r.emit(indent+padding, row.Spans)
			}
		}
	}
}
