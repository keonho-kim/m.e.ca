// Purpose: Parse agent Markdown into safe styled lines without HTML or terminal escape execution.
// Pattern: AST-to-terminal presentation adapter.
// Usage: Answers and Thinking share Render for completed blocks and streamed snapshots; file/tool bodies remain plain.
// Related: internal/tui/markdown/inline.go, internal/tui/markdown/table.go.
package markdown

import (
	"fmt"
	"github.com/rivo/uniseg"
	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/extension"
	extast "github.com/yuin/goldmark/v2/extension/ast"
	"strings"
)

type renderer struct {
	source []byte
	rows   []Line
	width  int
	failed bool
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
	r.blocks(document, "", 0)
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
	if len(r.rows) > 0 && r.rows[len(r.rows)-1].Text != "" {
		r.rows = append(r.rows, Line{})
	}
}
func (r *renderer) emit(prefix string, spans []Span) {
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
			r.rows = append(r.rows, line(current))
			current, text = nil, rest
		}
	}
	r.rows = append(r.rows, line(current))
}

func (r *renderer) blocks(parent ast.Node, indent string, depth int) {
	if depth > 128 {
		r.failed = true
		return
	}
	for n := parent.FirstChild(); n != nil; n = n.NextSibling() {
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
		index := v.Start
		for item := v.FirstChild(); item != nil; item = item.NextSibling() {
			marker := "• "
			if v.IsOrdered() {
				marker = fmt.Sprintf("%d. ", index)
				index++
			}
			if status, ok := extension.TaskStatusOf(item); ok {
				marker = "[ ] "
				if status == extension.TaskStatusCompleted {
					marker = "[x] "
				}
			}
			start := len(r.rows)
			continuation := indent + strings.Repeat(" ", uniseg.StringWidth(marker))
			r.blocks(item, continuation, depth+1)
			if len(r.rows) > start {
				first := r.rows[start]
				prefix := indent + marker
				// Replace only the structural indentation; keep content span styles.
				spans := first.Spans
				if len(spans) > 0 && strings.HasPrefix(spans[0].Text, continuation) {
					spans[0].Text = strings.TrimPrefix(spans[0].Text, continuation)
				}
				r.rows[start] = line(append([]Span{{Text: prefix, Style: Dim}}, spans...))
			}
			if v.IsTight {
				for len(r.rows) > start && r.rows[len(r.rows)-1].Text == "" {
					r.rows = r.rows[:len(r.rows)-1]
				}
			}
		}
		r.blank()
	case *ast.Blockquote:
		r.blocks(v, indent+"│ ", depth+1)
		r.blank()
	case *ast.CodeBlock:
		r.blank()
		if language, ok := v.Language(r.source); ok {
			r.emit(indent, []Span{{Text: SafeText(language), Style: Dim}})
		}
		body := strings.TrimSuffix(v.Value.Str(r.source), "\n")
		for _, part := range strings.Split(body, "\n") {
			r.emit(indent+"│ ", []Span{{Text: SafeText(part)}})
		}
		r.blank()
	case *ast.ThematicBreak:
		r.emit(indent, []Span{{Text: "────────", Style: Dim}})
		r.blank()
	case *extast.Table:
		r.table(v, indent)
		r.blank()
	case *ast.HTMLBlock:
		for _, part := range strings.Split(strings.TrimSuffix(v.Value.Str(r.source), "\n"), "\n") {
			r.emit(indent, []Span{{Text: SafeText(part)}})
		}
		r.blank()
	case *ast.LinkReferenceDefinition:
	// Its destination is displayed at the reference's actual use.
	default:
		r.blocks(n, indent, depth+1)
	}
}
