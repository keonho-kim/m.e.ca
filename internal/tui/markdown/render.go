// Purpose: Parse agent Markdown into safe styled lines without HTML or terminal escape execution.
// Pattern: AST-to-terminal presentation adapter.
// Usage: Answers, Thinking and Markdown file previews share Render; raw source and tool bodies remain plain.
// Related: internal/tui/markdown/inline.go, internal/tui/markdown/table.go, internal/tui/markdown/code_rows.go.
package markdown

import (
	"strings"

	"github.com/yuin/goldmark/v2/ast"
	extast "github.com/yuin/goldmark/v2/extension/ast"
)

type renderer struct {
	sink         *rowSink
	firstRow     func(Line) Line
	stopped      bool
	source       []byte
	rows         []Line
	width        int
	failed       bool
	listProgress *listProgress
	leadingBlank bool
}

// MaxParseBytes bounds AST construction, not accepted or displayed source text.
const MaxParseBytes = 16 * 1024 * 1024

// Render fits tables to width and leaves other blocks for viewport wrapping.
// Oversized replies remain literal text instead of building an unbounded AST.
func Render(text string, width int) []Line {
	if width < 1 {
		return nil
	}
	if len(text) > MaxParseBytes {
		return literal(text)
	}
	r := renderer{source: []byte(text), width: width}
	document := documentParser.Parse(r.source)
	r.document(document)
	if r.failed {
		return literal(text)
	}
	for len(r.rows) > 0 && r.rows[len(r.rows)-1].Text == "" {
		r.rows = r.rows[:len(r.rows)-1]
	}
	for count := len(text) - len(strings.TrimRight(text, "\n")); count > 0; count-- {
		r.rows = append(r.rows, Line{})
	}
	return r.rows
}
func literal(text string) []Line {
	var rows []Line
	for _, part := range strings.Split(SafeText(text), "\n") {
		rows = append(rows, line([]Span{{Text: part}}))
	}
	return rows
}
func (r *renderer) blank() {
	if r.rowCount() == 0 {
		r.leadingBlank = true
	}
	if r.rowCount() > 0 && !r.lastBlank() {
		r.appendRow(Line{})
	}
}
func (r *renderer) emit(prefix string, spans []Span) {
	if r.stopped || r.failed {
		return
	}
	// Explicit hard breaks produce real rows before terminal-width wrapping.
	var current []Span
	current = add(current, prefix, Dim)
	for _, span := range spans {
		text := span.Text
		for {
			chunk, rest, more := strings.Cut(text, "\n")
			current = add(current, chunk, span.Style)
			if !more {
				break
			}
			r.appendRow(line(current))
			current, text = nil, rest
		}
	}
	r.appendRow(line(current))
}

func (r *renderer) blocks(parent ast.Node, indent string, depth int) {
	if depth > 128 {
		r.failed = true
		return
	}
	for n := parent.FirstChild(); n != nil; n = n.NextSibling() {
		if r.failed || r.stopped {
			return
		}
		r.block(n, indent, depth)
	}
}

func (r *renderer) block(n ast.Node, indent string, depth int) {
	switch v := n.(type) {
	case *ast.Heading:
		r.blank()
		r.emit(indent, r.inline(v, Bold, 0))
		r.blank()
	case *ast.Paragraph:
		r.emit(indent, r.inline(n, 0, 0))
		r.blank()
	case *ast.List:
		r.list(v, indent, depth)
	case *ast.Blockquote:
		r.blocks(v, indent+"│ ", depth+1)
		r.blank()
	case *ast.CodeBlock:
		r.blank()
		if language, ok := v.Language(r.source); ok {
			r.emit(indent, []Span{{Text: SafeText(language), Style: Dim}})
		}
		body := strings.TrimSuffix(v.Value.Str(r.source), "\n")
		prefix := indent + codeMarker
		empty := line([]Span{{Text: prefix, Style: Dim}})
		for part := range strings.SplitSeq(body, "\n") {
			if r.failed || r.stopped {
				return
			}
			if part == "" && r.sink != nil {
				r.appendRow(empty)
			} else {
				r.appendRow(codeLine(prefix, part))
			}
		}
		r.blank()
	case *ast.ThematicBreak:
		r.emit(indent, []Span{{Text: "────────", Style: Dim}})
		r.blank()
	case *extast.Table:
		r.table(v, indent)
		r.blank()
	case *ast.HTMLBlock:
		for part := range strings.SplitSeq(strings.TrimSuffix(v.Value.Str(r.source), "\n"), "\n") {
			if r.failed || r.stopped {
				return
			}
			r.emit(indent, []Span{{Text: SafeText(part)}})
		}
		r.blank()
	case *ast.LinkReferenceDefinition:
	// Its destination is displayed at the reference's actual use.
	default:
		r.blocks(n, indent, depth+1)
	}
}
