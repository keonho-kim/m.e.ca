// Purpose: Resume oversized Markdown windows from bounded immutable rendered-block plans.
// Pattern: Source-backed code checkpoints and compact ordinary blocks without retained ASTs.
// Usage: Viewport uses RenderResumable when full compact rows exceed its display cache budget.
// Related: internal/tui/markdown/code_rows.go, internal/tui/markdown/render_packed.go.
package markdown

import (
	"sort"
	"strings"
	"unsafe"

	"github.com/yuin/goldmark/v2/ast"
)

// Plan owns only derived display resources. Borrowed code text stays with its immutable source owner.
type Plan struct {
	blocks         []planBlock
	rows, retained int
}
type planBlock struct {
	start, rows int
	packed      *Packed
	code        *codeRows
}

// RenderResumable parses once and chooses compact complete rows or a resumable
// plan. Both nil means the caller should use the existing uncached row visitor.
func RenderResumable(text string, width, budget int) (*Packed, *Plan) {
	if width < 1 || budget < 1 || len(text) > MaxParseBytes {
		return nil, nil
	}
	source := []byte(text)
	document := documentParser.Parse(source)
	if packed := renderPackedDocument(text, source, document, width, budget); packed != nil {
		return packed, nil
	}
	return nil, newPlan(text, source, document, width, budget)
}

func newPlan(text string, source []byte, document ast.Node, width, budget int) *Plan {
	p := &Plan{}
	pending := 0
	resources := 0
	for node := document.FirstChild(); node != nil; node = node.NextSibling() {
		block := planBlock{}
		leading, trailing := false, 0
		if code, ok := node.(*ast.CodeBlock); ok {
			block.code = newCodeRows(text, source, code, budget-resources)
			if block.code == nil {
				return nil
			}
			block.rows = block.code.rows
			resources += block.code.retained
			leading, trailing = true, 1
		} else {
			group := collectNodes(node, node.NextSibling(), source, width, budget-resources)
			if group == nil {
				return nil
			}
			group.value.text = group.text.String()
			group.value.retained = group.weight()
			packed := group.value
			block.packed = &packed
			block.rows = group.value.Len()
			leading, trailing = group.leading, group.pending
		}
		if leading && p.rows > 0 && pending == 0 {
			pending = 1
		}
		if block.rows == 0 {
			pending += trailing
			continue
		}
		if block.packed != nil {
			resources += block.packed.retained
		}
		if pending > 0 {
			p.add(planBlock{rows: pending})
			pending = 0
		}
		p.add(block)
		pending = trailing
		p.retained = resources + int(unsafe.Sizeof(*p)) + cap(p.blocks)*int(unsafe.Sizeof(planBlock{}))
		if p.retained > budget {
			return nil
		}
	}
	if trailing := len(text) - len(strings.TrimRight(text, "\n")); trailing > 0 {
		p.add(planBlock{rows: trailing})
	}
	p.retained = resources + int(unsafe.Sizeof(*p)) + cap(p.blocks)*int(unsafe.Sizeof(planBlock{}))
	if p.retained > budget {
		return nil
	}
	return p
}

func (p *Plan) add(block planBlock) {
	block.start = p.rows
	p.rows += block.rows
	p.blocks = append(p.blocks, block)
}

// RetainedBytes counts derived resources; borrowed source stays with its caller.
func (p *Plan) RetainedBytes() int { return p.retained }

// Walk resumes at a logical rendered row and stops immediately when visit returns false.
func (p *Plan) Walk(start int, visit func(Line) bool) {
	start = max(0, start)
	first := sort.Search(len(p.blocks), func(index int) bool { return p.blocks[index].start+p.blocks[index].rows > start })
	for _, block := range p.blocks[first:] {
		offset := max(0, start-block.start)
		if block.code != nil {
			if !block.code.walk(offset, visit) {
				return
			}
			continue
		}
		for row := offset; row < block.rows; row++ {
			value := Line{}
			if block.packed != nil {
				value = packedLogicalLine(block.packed, row)
			}
			if !visit(value) {
				return
			}
		}
	}
}

func packedLogicalLine(p *Packed, index int) Line {
	index = p.position(index)
	var base, first uint32
	if index > 0 {
		base, first = p.lines[index-1].textEnd, p.lines[index-1].spanEnd
	}
	end := p.lines[index]
	var spans []Span
	if end.spanEnd > first {
		spans = make([]Span, 0, end.spanEnd-first)
	}
	at := base
	for _, span := range p.spans[first:end.spanEnd] {
		spans = append(spans, Span{Text: p.text[at:span.end], Style: span.style})
		at = span.end
	}
	return Line{Text: p.text[base:end.textEnd], Spans: spans}
}
