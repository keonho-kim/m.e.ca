// Purpose: Materialize only the visible rows from a prepared sparse block index.
// Pattern: Indexed presentation window.
// Usage: Transcript scrolling and copying request bounded block windows.
// Related: internal/tui/viewport/block_layout.go, internal/tui/viewport/cache_budget.go.
package viewport

import (
	"slices"
	"sort"

	"github.com/keonho-kim/meca/internal/tui/textlayout"
)

func (b *Block) Window(width, start, count int) []Line {
	return b.appendWindow(nil, width, start, count)
}

// appendWindow writes into the caller's visible-row buffer. Existing rows keep
// their ownership and styling; standalone Window calls get an independent slice.
func (b *Block) appendWindow(out []Line, width, start, count int) []Line {
	start = max(0, start)
	if count < 1 || start >= b.Count(width) {
		return out
	}
	l := &b.layout
	start = max(0, start) + l.trim
	end := min(l.total, start+count)
	sourceEnd := l.total
	if b.SpaceAfter > 0 {
		sourceEnd -= l.trailing
	}
	renderEnd := min(end, sourceEnd)
	first := len(out)
	out = slices.Grow(out, end-start)
	if start < renderEnd {
		if l.markdown && b.ensureMarkdown(width) && !b.live {
			for _, row := range l.parsed {
				b.retainParsedLine(row)
			}
		}
		if l.markdown && b.cacheBytes() > displayCacheBudget {
			b.compactMarkdown()
		}
		at := sort.Search(len(l.points), func(i int) bool { return l.points[i].row > start }) - 1
		point := l.points[max(0, at)]
		row := point.row
		appendRow := func(value Line) bool {
			if row >= start {
				value.ID, value.Row, value.Thinking = b.ID, row, b.Thinking
				out = append(out, value)
			}
			row++
			return row < renderEnd
		}
		if l.markdown {
			for i := point.source; i < l.markdownLines() && row < renderEnd; i++ {
				offset, prefix := 0, ""
				if i == point.source {
					offset, prefix = point.offset, point.prefix
				}
				// Concrete iterators keep normal row wrapping on the stack.
				if l.packed != nil {
					for wrapped := range l.packed.RowsFrom(i, width, offset, prefix) {
						if !appendRow(Line{Text: wrapped.Text, Spans: wrapped.Spans, Copy: wrapped.Copy}) {
							break
						}
					}
				} else {
					for wrapped := range l.parsed[i].RowsFrom(width, offset, prefix) {
						if !appendRow(Line{Text: wrapped.Text, Spans: wrapped.Spans, Copy: wrapped.Copy}) {
							break
						}
					}
				}
			}
		} else {
			for wrapped := range textlayout.RowsFrom(l.source, width, point.source, point.prefix) {
				if !appendRow(Line{Text: wrapped.Text, Copy: textlayout.Fragment{Source: l.source, Row: wrapped}}) {
					break
				}
			}
			b.classifyChanges(out[first:])
		}
	}
	for row := max(start, sourceEnd); row < end; row++ {
		out = append(out, Line{ID: b.ID, Row: row, Thinking: b.Thinking})
	}
	if b.Centered {
		centerLines(out[first:], width)
	}
	// The visible rows belong to the transcript window. Completed blocks keep
	// compact rows when they fit; otherwise sparse indexes support restoration.
	// Live blocks keep incremental prefixes within the same shared budget.
	if !b.live {
		if b.cacheBytes() > displayCacheBudget {
			b.releaseParsed()
		}
	}
	return out
}
