// Purpose: Reuse complete lines in a top-level open fenced code block.
// Pattern: Parser checkpoint with exact fallback at closing fences and changed prefixes.
// Usage: Stream keeps one checkpoint after a normal parse and renders only its mutable suffix.
// Related: internal/tui/markdown/stream.go, internal/tui/markdown/render.go.
package markdown

import (
	"strings"

	"github.com/yuin/goldmark/v2/ast"
)

type fenceCheckpoint struct {
	cut, rows, first, weight int
	marker                   byte
	size                     int
}

func openFence(node ast.Node, text string, start int, source []byte, endRows int) *fenceCheckpoint {
	code, ok := node.(*ast.CodeBlock)
	if !ok || code.CodeBlockKind != ast.CodeBlockKindFenced {
		return nil
	}
	at := start + blockStart(node, source)
	line, body, more := strings.Cut(text[at:], "\n")
	if !more || len(line) < 3 || (line[0] != '`' && line[0] != '~') {
		return nil
	}
	size := 0
	for size < len(line) && line[size] == line[0] {
		size++
	}
	if size < 3 {
		return nil
	}
	p := &fenceCheckpoint{marker: line[0], size: size}
	if p.closed(body) {
		return nil
	}
	first := endRows - 1 - (strings.Count(strings.TrimSuffix(body, "\n"), "\n") + 1)
	if first < 0 {
		return nil
	}
	p.first, p.rows = first, first+strings.Count(body, "\n")
	p.cut = at + len(line) + 1 + strings.LastIndexByte(body, '\n') + 1
	return p
}

func (p *fenceCheckpoint) closed(source string) bool {
	for {
		line, rest, more := strings.Cut(source, "\n")
		trimmed := strings.TrimLeft(line, " ")
		if len(line)-len(trimmed) < 4 {
			size := 0
			for size < len(trimmed) && trimmed[size] == p.marker {
				size++
			}
			if size >= p.size && strings.TrimSpace(trimmed[size:]) == "" {
				return true
			}
		}
		if !more {
			return false
		}
		source = rest
	}
}

func (s *Stream) renderFence(text string, width int) ([]Line, bool) {
	p := s.fence
	if len(text) > MaxParseBytes || p.cut > len(text) || p.cut > len(s.source) || text[:p.cut] != s.source[:p.cut] {
		return nil, false
	}
	suffix := text[p.cut:]
	if p.closed(suffix) {
		return nil, false
	}
	r := renderer{width: width}
	if suffix != "" || p.rows == p.first {
		for _, part := range strings.Split(strings.TrimSuffix(suffix, "\n"), "\n") {
			r.emit("│ ", []Span{{Text: SafeText(part)}})
		}
	}
	before, weight := p.rows, p.weight
	stable := strings.Count(suffix, "\n")
	for i, row := range r.rows {
		weight += lineWeight(row)
		if i+1 == stable {
			p.weight = weight
		}
	}
	if stable > 0 {
		p.rows += stable
		p.cut += strings.LastIndexByte(suffix, '\n') + 1
	}
	previous := s.rows
	s.rows = append(s.rows[:before], r.rows...)
	s.rows = append(s.rows, Line{})
	if len(previous) > len(s.rows) {
		clear(previous[len(s.rows):])
	}
	s.source, s.weight, s.reused = text, weight, before
	return s.normalized(text), true
}
