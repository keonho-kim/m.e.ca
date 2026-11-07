// Purpose: Keep rendered logical Markdown rows in bounded text, style and repeated-row ranges.
// Pattern: Immutable compact presentation snapshot.
// Usage: Viewport packs oversized rendered rows and materializes only visible wrapped rows.
// Related: internal/tui/markdown/layout.go, internal/tui/markdown/styles.go.
package markdown

import (
	"iter"
	"sort"
)

type packedLine struct{ textEnd, spanEnd, through uint32 }
type packedSpan struct {
	end   uint32
	style Style
}

// Packed owns rendered text, never an AST or a second source-document copy.
type Packed struct {
	text     string
	lines    []packedLine
	spans    []packedSpan
	retained int
}

// Pack accepts complete logical Render rows. An unsuitable or oversized cache
// returns nil; callers still display every row through their ordinary renderer.
func Pack(lines []Line, budget int) *Packed {
	p := &packedCollector{budget: budget}
	if budget < 1 || !p.add(lines) || !p.blanks() || len(p.value.lines) == 0 {
		return nil
	}
	p.value.text = p.text.String()
	p.value.retained = p.weight()
	return &p.value
}

func (p *Packed) Len() int {
	if len(p.lines) == 0 {
		return 0
	}
	return int(p.lines[len(p.lines)-1].through)
}
func (p *Packed) RetainedBytes() int { return p.retained }

func (p *Packed) position(index int) int {
	if p.Len() == len(p.lines) {
		return index
	}
	return sort.Search(len(p.lines), func(i int) bool { return p.lines[i].through > uint32(index) })
}

// Text returns one immutable logical row for sparse wrapping indexes.
func (p *Packed) Text(index int) string {
	index = p.position(index)
	var start uint32
	if index > 0 {
		start = p.lines[index-1].textEnd
	}
	return p.text[start:p.lines[index].textEnd]
}

// RowsFrom reconstructs spans only for the requested window, including lines
// with many style runs. Copy offsets stay relative to the same rendered line.
func (p *Packed) RowsFrom(index, width, start int, prefix string) iter.Seq[Line] {
	return func(yield func(Line) bool) {
		index := p.position(index)
		var base, first uint32
		if index > 0 {
			base, first = p.lines[index-1].textEnd, p.lines[index-1].spanEnd
		}
		end := p.lines[index]
		runs := p.spans[first:end.spanEnd]
		at := sort.Search(len(runs), func(i int) bool { return int(runs[i].end) > int(base)+start })
		at = min(at, max(0, len(runs)-1))
		cursor := spanCursor{packed: runs, index: at, base: int(base)}
		for row := range styledRows(p.text[base:end.textEnd], width, start, prefix, cursor) {
			if !yield(row) {
				return
			}
		}
	}
}
