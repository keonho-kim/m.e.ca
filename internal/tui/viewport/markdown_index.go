// Purpose: Extend Markdown row metrics and sparse positions while reusing immutable stream prefixes.
// Pattern: Incremental block index builder.
// Usage: Block.prepare delegates Markdown indexing after selecting its parsed rows.
// Related: internal/tui/viewport/block_layout.go, internal/tui/viewport/cache_budget.go.
package viewport

import (
	"sort"
)

func (b *Block) indexMarkdown(previous blockLayout, width int) (firstContent, lastContent int) {
	l := &b.layout
	if l.sparse {
		return b.indexMarkdownWalk(width)
	}
	firstContent, lastContent = -1, -1
	reused := 0
	if b.live && l.packed == nil && previous.key.width == width {
		reused = min(b.stream.ReusedRows(), len(previous.metrics), len(l.parsed))
	}
	l.metrics = previous.metrics[:reused]
	if !b.live || l.packed != nil {
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
		b.indexMarkdownText(text, i, width, &firstContent, &lastContent)
	}
	return firstContent, lastContent
}
