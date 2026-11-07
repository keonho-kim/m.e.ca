// Purpose: Collect bounded compact rows and coalesce identical consecutive rendered lines.
// Pattern: Immutable row encoding shared by full-row packing and streaming rendering.
// Usage: Pack and RenderPacked own one collector and publish only its Packed value.
// Related: internal/tui/markdown/packed.go, internal/tui/markdown/render_packed.go.
package markdown

import (
	"strings"
	"unsafe"
)

type packedCollector struct {
	value           Packed
	text            strings.Builder
	budget, pending int
	leading         bool
}

func (p *packedCollector) weight() int {
	return int(unsafe.Sizeof(Packed{})) + p.text.Cap() + cap(p.value.lines)*int(unsafe.Sizeof(packedLine{})) + cap(p.value.spans)*int(unsafe.Sizeof(packedSpan{}))
}

func (p *packedCollector) blanks() bool {
	if p.pending > 0 {
		p.value.lines = append(p.value.lines, packedLine{uint32(p.text.Len()), uint32(len(p.value.spans)), uint32(p.value.Len() + p.pending)})
		p.pending = 0
		if p.weight() > p.budget {
			return false
		}
	}
	return true
}

func (p *packedCollector) add(lines []Line) bool {
	for _, line := range lines {
		if line.Text == "" {
			p.pending++
			continue
		}
		if !p.blanks() {
			return false
		}
		if p.sameLast(line) {
			p.value.lines[len(p.value.lines)-1].through++
			continue
		}
		base, at := p.text.Len(), 0
		p.text.WriteString(line.Text)
		for _, span := range line.Spans {
			end := at + len(span.Text)
			if end > len(line.Text) || line.Text[at:end] != span.Text {
				return false
			}
			p.value.spans = append(p.value.spans, packedSpan{uint32(base + end), span.Style})
			at = end
		}
		if at != len(line.Text) {
			return false
		}
		p.value.lines = append(p.value.lines, packedLine{uint32(p.text.Len()), uint32(len(p.value.spans)), uint32(p.value.Len() + 1)})
		if p.weight() > p.budget {
			return false
		}
	}
	return true
}

func (p *packedCollector) sameLast(line Line) bool {
	count := len(p.value.lines)
	if count == 0 {
		return false
	}
	var start, first uint32
	if count > 1 {
		start, first = p.value.lines[count-2].textEnd, p.value.lines[count-2].spanEnd
	}
	last := p.value.lines[count-1]
	if p.text.String()[start:last.textEnd] != line.Text || int(last.spanEnd-first) != len(line.Spans) {
		return false
	}
	at := start
	for i, span := range line.Spans {
		stored := p.value.spans[int(first)+i]
		if stored.style != span.Style || int(stored.end-at) != len(span.Text) || p.text.String()[at:stored.end] != span.Text {
			return false
		}
		at = stored.end
	}
	return true
}

// join preserves group boundary blanks while copying only compact text/ranges.
func (p *packedCollector) join(group *packedCollector) bool {
	if group.leading && len(p.value.lines) > 0 && p.pending == 0 {
		p.pending = 1
	}
	if len(group.value.lines) == 0 {
		p.pending += group.pending
		return true
	}
	if !p.blanks() {
		return false
	}
	base, spans := uint32(p.text.Len()), uint32(len(p.value.spans))
	through := uint32(p.value.Len())
	p.text.WriteString(group.text.String())
	for _, row := range group.value.lines {
		p.value.lines = append(p.value.lines, packedLine{row.textEnd + base, row.spanEnd + spans, row.through + through})
	}
	for _, span := range group.value.spans {
		p.value.spans = append(p.value.spans, packedSpan{span.end + base, span.style})
	}
	p.pending = group.pending
	return p.weight() <= p.budget
}
