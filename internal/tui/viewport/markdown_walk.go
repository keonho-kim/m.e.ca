// Purpose: Index and resume oversized Markdown through bounded plans or uncached row walking.
// Pattern: Bounded renderer visitor over immutable block source.
// Usage: Blocks whose compact rows exceed the cache budget use Walk without full row arrays.
// Related: internal/tui/markdown/walk.go, internal/tui/viewport/markdown_index.go, internal/tui/viewport/block_window.go.
package viewport

import (
	"strings"

	"github.com/keonho-kim/meca/internal/tui/markdown"
	"github.com/keonho-kim/meca/internal/tui/textlayout"
)

func (b *Block) indexMarkdownText(text string, index, width int, first, last *int) {
	l := &b.layout
	if index%256 == 0 {
		l.points = append(l.points, blockPoint{source: index, row: l.total, last: *last})
	}
	var metric rowMetrics
	for row := range textlayout.Rows(text, width) {
		if metric.count > 0 && metric.count%256 == 0 {
			l.points = append(l.points, blockPoint{source: index, row: l.total + metric.count, offset: row.Start, within: metric.count, prefix: row.Prefix, last: *last})
		}
		if strings.TrimSpace(row.Text) == "" {
			if metric.leading == metric.count {
				metric.leading++
			}
			metric.trailing++
		} else {
			metric.trailing = 0
		}
		metric.count++
	}
	if b.live && !l.sparse && l.packed == nil {
		l.metrics = append(l.metrics, metric)
	}
	if metric.leading < metric.count {
		if *first < 0 {
			*first = l.total + metric.leading
		}
		*last = l.total + metric.count - metric.trailing - 1
	}
	l.total += metric.count
}

func (b *Block) indexMarkdownWalk(width int) (first, last int) {
	first, last, index := -1, -1, 0
	visit := func(row markdown.Line) bool {
		b.indexMarkdownText(row.Text, index, width, &first, &last)
		index++
		return true
	}
	valid := true
	if b.layout.plan != nil {
		b.layout.plan.Walk(0, visit)
	} else {
		valid = markdown.Walk(b.layout.source, width, visit)
	}
	if !valid {
		l := &b.layout
		l.markdown, l.sparse = false, false
		l.total, l.points, l.metrics = 0, nil, nil
		l.source = markdown.SafeText(l.source)
		return b.indexLiteral(width)
	}
	return first, last
}

func (b *Block) walkMarkdownWindow(point blockPoint, width int, appendRow func(Line) bool) {
	index := 0
	visit := func(row markdown.Line) bool {
		current := index
		index++
		if current < point.source {
			return true
		}
		offset, prefix := 0, ""
		if current == point.source {
			offset, prefix = point.offset, point.prefix
		}
		for wrapped := range row.RowsFrom(width, offset, prefix) {
			if !appendRow(Line{Text: wrapped.Text, Spans: wrapped.Spans, Copy: wrapped.Copy}) {
				return false
			}
		}
		return true
	}
	if b.layout.plan != nil {
		index = point.source
		b.layout.plan.Walk(point.source, visit)
	} else {
		markdown.Walk(b.layout.source, width, visit)
	}
}
