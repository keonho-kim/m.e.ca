// Purpose: Materialize only the visible rows from a prepared sparse block index.
// Pattern: Indexed presentation window.
// Usage: Transcript scrolling and copying request bounded block windows.
// Related: internal/tui/viewport/block_layout.go, internal/tui/viewport/cache_budget.go.
package viewport

import (
	"sort"

	"github.com/keonho-kim/meca/internal/tui/textlayout"
)

func (b *Block) Window(width, start, count int) []Line {
	start = max(0, start)
	if count < 1 || start >= b.Count(width) {
		return nil
	}
	l := &b.layout
	if l.markdown && b.ensureMarkdown(width) && !b.live {
		for _, row := range l.parsed {
			b.retainParsedLine(row)
		}
	}
	start = max(0, start) + l.trim
	end := min(l.total, start+count)
	at := sort.Search(len(l.points), func(i int) bool { return l.points[i].row > start }) - 1
	point := l.points[max(0, at)]
	row := point.row
	out := make([]Line, 0, end-start)
	appendRow := func(value Line) bool {
		if row >= start {
			value.ID, value.Row, value.Thinking = b.ID, row, b.Thinking
			out = append(out, value)
		}
		row++
		return row < end
	}
	if l.markdown {
		for i := point.source; i < len(l.parsed) && row < end; i++ {
			offset, prefix := 0, ""
			if i == point.source {
				offset, prefix = point.offset, point.prefix
			}
			for wrapped := range l.parsed[i].RowsFrom(width, offset, prefix) {
				if !appendRow(Line{Text: wrapped.Text, Spans: wrapped.Spans, Copy: wrapped.Copy}) {
					break
				}
			}
		}
	} else {
		for wrapped := range textlayout.RowsFrom(l.source, width, point.source, point.prefix) {
			if !appendRow(Line{Text: wrapped.Text, Copy: textlayout.Fragment{Source: l.source, Row: wrapped}}) {
				break
			}
		}
		b.classifyChanges(out)
	}
	if b.Centered {
		centerLines(out, width)
	}
	// The visible rows belong to the transcript window. Large immutable blocks
	// need only their sparse index between scrolls, while live blocks retain the
	// incremental prefix within the transcript's shared display-cache budget.
	if !b.live {
		if b.cacheBytes() > displayCacheBudget {
			b.releaseParsed()
		}
	}
	return out
}
