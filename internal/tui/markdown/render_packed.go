// Purpose: Collect rendered Markdown rows within a bounded compact representation.
// Pattern: Incremental collection through the shared renderer row sink.
// Usage: RenderResumable tries the shared AST collector before a bounded block plan.
// Related: internal/tui/markdown/render.go, internal/tui/markdown/packed.go, internal/tui/markdown/plan.go.
package markdown

import (
	"strings"
	"sync"
	"unsafe"

	"github.com/yuin/goldmark/v2/ast"
)

func renderPackedDocument(text string, source []byte, document ast.Node, width, budget int) *Packed {
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
	p := &packedCollector{budget: budget}
	r := renderer{source: source, width: width, sink: &rowSink{send: func(value Line) bool { return p.add([]Line{value}) }}}
	for node := first; node != nil && node != until; node = node.NextSibling() {
		r.block(node, "", 0)
		if r.failed || r.stopped {
			return nil
		}
	}
	p.pending += r.sink.pending
	p.leading = r.leadingBlank
	return p
}
