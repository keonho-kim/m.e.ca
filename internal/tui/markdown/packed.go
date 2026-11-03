// Purpose: Keep completed logical Markdown rows in bounded flat text and style arrays.
// Pattern: Immutable compact presentation snapshot.
// Usage: Viewport packs oversized rendered rows and materializes only visible wrapped rows.
// Related: internal/tui/markdown/layout.go, internal/tui/markdown/styles.go.
package markdown

import (
	"iter"
	"sort"
	"strings"
	"unsafe"
)

type packedLine struct{ textEnd, spanEnd uint32 }
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
	textBytes, spans := 0, 0
	size := int(unsafe.Sizeof(Packed{}))
	for _, line := range lines {
		textBytes += len(line.Text)
		spans += len(line.Spans)
		size += int(unsafe.Sizeof(packedLine{})) + len(line.Text) + len(line.Spans)*int(unsafe.Sizeof(packedSpan{}))
		if size > budget || uint64(textBytes) > uint64(^uint32(0)) || uint64(spans) > uint64(^uint32(0)) {
			return nil
		}
	}
	if len(lines) == 0 || size > budget {
		return nil
	}
	p := &Packed{lines: make([]packedLine, len(lines)), spans: make([]packedSpan, spans)}
	var text strings.Builder
	text.Grow(textBytes)
	spanAt := 0
	for i, line := range lines {
		base, at := text.Len(), 0
		text.WriteString(line.Text)
		for _, span := range line.Spans {
			end := at + len(span.Text)
			if end > len(line.Text) || line.Text[at:end] != span.Text {
				return nil
			}
			p.spans[spanAt] = packedSpan{end: uint32(base + end), style: span.Style}
			spanAt++
			at = end
		}
		if at != len(line.Text) {
			return nil
		}
		p.lines[i] = packedLine{uint32(text.Len()), uint32(spanAt)}
	}
	p.text = text.String()
	p.retained = size - textBytes + text.Cap()
	if p.retained > budget {
		return nil
	}
	return p
}

func (p *Packed) Len() int           { return len(p.lines) }
func (p *Packed) RetainedBytes() int { return p.retained }

// Text returns one immutable logical row for sparse wrapping indexes.
func (p *Packed) Text(index int) string {
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
