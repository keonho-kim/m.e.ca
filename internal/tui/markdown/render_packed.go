// Purpose: Collect completed Markdown into compact rows while recycling block scratch space.
// Pattern: Incremental collection over the existing Markdown renderer.
// Usage: Large static viewport blocks use RenderPacked; unsupported budgets fall back to Render.
// Related: internal/tui/markdown/render.go, internal/tui/markdown/packed.go.
package markdown

import (
	"strings"
	"sync"
	"unsafe"

	"github.com/yuin/goldmark/v2/ast"
)

// RenderPacked preserves Render semantics while avoiding a full-document Line
// array. Lists and tables keep their own native rendering before collection.
func RenderPacked(text string, width, budget int) *Packed {
	if width < 1 || len(text) > MaxParseBytes || budget < 1 {
		return nil
	}
	source := []byte(text)
	document := documentParser.Parse(source)
	starts, parallel := partitionDocument(document, len(source))
	var p *packedCollector
	if !parallel {
		p = collectNodes(document.FirstChild(), nil, source, width, budget)
	} else {
		var groups [4]*packedCollector
		var workers sync.WaitGroup
		for i := range groups {
			workers.Go(func() { groups[i] = collectNodes(starts[i], starts[i+1], source, width, budget/len(groups)) })
		}
		workers.Wait()
		p = &packedCollector{budget: budget}
		rows, spans, textBytes := 0, 0, 0
		for _, group := range groups {
			if group == nil {
				return nil
			}
			rows += len(group.value.lines) + group.pending + 1
			spans += len(group.value.spans)
			textBytes += group.text.Len()
		}
		if int(unsafe.Sizeof(Packed{}))+textBytes+rows*int(unsafe.Sizeof(packedLine{}))+spans*int(unsafe.Sizeof(packedSpan{})) > budget {
			return nil
		}
		p.text.Grow(textBytes)
		p.value.lines = make([]packedLine, 0, rows)
		p.value.spans = make([]packedSpan, 0, spans)
		for i, group := range groups {
			if !p.join(group) {
				return nil
			}
			groups[i] = nil
		}
	}
	if p == nil {
		return nil
	}
	// Render removes derived trailing blanks, then restores source newlines.
	p.pending = len(text) - len(strings.TrimRight(text, "\n"))
	if !p.blanks() || len(p.value.lines) == 0 {
		return nil
	}
	p.value.text = p.text.String()
	p.value.retained = p.weight()
	return &p.value
}

func collectNodes(first, until ast.Node, source []byte, width, budget int) *packedCollector {
	r := renderer{source: source, width: width}
	p := &packedCollector{budget: budget}
	for node := first; node != nil && node != until; node = node.NextSibling() {
		r.block(node, "", 0)
		if r.failed {
			return nil
		}
		if len(r.rows) >= 512 {
			// The preceding row owns blank insertion across block boundaries.
			end := len(r.rows) - 1
			if !p.add(r.rows[:end]) {
				return nil
			}
			last := r.rows[end]
			if cap(r.rows) > 2048 {
				r.rows = []Line{last}
			} else {
				clear(r.rows)
				r.rows = append(r.rows[:0], last)
			}
		}
	}
	if !p.add(r.rows) {
		return nil
	}
	p.leading = r.leadingBlank
	return p
}

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
	for p.pending > 0 {
		p.value.lines = append(p.value.lines, packedLine{uint32(p.text.Len()), uint32(len(p.value.spans))})
		p.pending--
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
		p.value.lines = append(p.value.lines, packedLine{uint32(p.text.Len()), uint32(len(p.value.spans))})
		if p.weight() > p.budget {
			return false
		}
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
	p.text.WriteString(group.text.String())
	for _, row := range group.value.lines {
		p.value.lines = append(p.value.lines, packedLine{row.textEnd + base, row.spanEnd + spans})
	}
	for _, span := range group.value.spans {
		p.value.spans = append(p.value.spans, packedSpan{span.end + base, span.style})
	}
	p.pending = group.pending
	return p.weight() <= p.budget
}
