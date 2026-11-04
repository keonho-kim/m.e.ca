// Purpose: Extend Markdown row metrics and sparse positions while reusing immutable stream prefixes.
// Pattern: Incremental block index builder.
// Usage: Block.prepare delegates Markdown indexing after selecting its parsed rows.
// Related: internal/tui/viewport/block_layout.go, internal/tui/viewport/cache_budget.go.
package viewport

import (
	"sort"
	"strings"

	"github.com/keonho-kim/meca/internal/tui/textlayout"
)

func (b *Block) indexMarkdown(previous blockLayout, width int) (firstContent, lastContent int) {
	l := &b.layout
	firstContent, lastContent = -1, -1
	reused := 0
	if b.live && previous.key.width == width {
		reused = min(b.stream.ReusedRows(), len(previous.metrics), len(l.parsed))
	}
	l.metrics = previous.metrics[:reused]
	if !b.live {
		// Completed windows resume from sparse points; only a live prefix
		// needs one reusable metric for every rendered logical row.
		l.metrics = nil
	}
	l.points = previous.points[:0]
	if reused > 0 {
		kept := sort.Search(len(previous.points), func(i int) bool { return previous.points[i].source >= reused })
		point := previous.points[kept-1]
		l.points = previous.points[:kept]
		l.total, lastContent = point.row-point.within, point.last
		for _, metric := range l.metrics[point.source:] {
			if metric.leading < metric.count {
				lastContent = l.total + metric.count - metric.trailing - 1
			}
			l.total += metric.count
		}
		if previous.first >= 0 && previous.first < l.total {
			firstContent = previous.first
		}
	}
	for i := reused; i < l.markdownLines(); i++ {
		var text string
		if l.packed != nil {
			text = l.packed.Text(i)
		} else {
			line := l.parsed[i]
			b.retainParsedLine(line)
			text = line.Text
		}
		if i%256 == 0 {
			l.points = append(l.points, blockPoint{source: i, row: l.total, last: lastContent})
		}
		var metric rowMetrics
		for row := range textlayout.Rows(text, width) {
			if metric.count > 0 && metric.count%256 == 0 {
				l.points = append(l.points, blockPoint{source: i, row: l.total + metric.count, offset: row.Start, within: metric.count, prefix: row.Prefix, last: lastContent})
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
		if b.live {
			l.metrics = append(l.metrics, metric)
		}
		if metric.leading < metric.count {
			if firstContent < 0 {
				firstContent = l.total + metric.leading
			}
			lastContent = l.total + metric.count - metric.trailing - 1
		}
		l.total += metric.count
	}
	return firstContent, lastContent
}
