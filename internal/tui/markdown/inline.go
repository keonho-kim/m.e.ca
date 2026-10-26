// Purpose: Convert inline Markdown nodes to safe monochrome text spans.
// Pattern: AST presentation projection.
// Usage: Block and table renderers call inline for their textual children.
// Related: internal/tui/markdown/render.go, internal/tui/markdown/layout.go.
package markdown

import (
	"github.com/yuin/goldmark/v2/ast"
	"strings"
)

func (r *renderer) inline(n ast.Node, style Style, depth int) []Span {
	var out inlineText
	r.collectInline(n, style, depth, &out)
	out.flush()
	return out.spans
}

func (r *renderer) collectInline(n ast.Node, style Style, depth int, out *inlineText) {
	if depth > 128 {
		r.failed = true
		return
	}
	for node := n.FirstChild(); node != nil; node = node.NextSibling() {
		switch v := node.(type) {
		case *ast.Text:
			out.add(SafeText(v.Value.Value(r.source)), style)
			if v.HardLineBreak() {
				out.add("\n", style)
			} else if v.SoftLineBreak() {
				out.add(" ", style)
			}
		case *ast.CodeSpan:
			out.add(SafeText(v.Value.Value(r.source)), style|Underline)
		case *ast.Strong:
			r.collectInline(v, style|Bold, depth+1, out)
		case *ast.Emphasis:
			r.collectInline(v, style|Italic, depth+1, out)
		case *ast.Link:
			r.collectInline(v, style|Underline, depth+1, out)
			out.add(" ("+SafeText(v.Destination.Value(r.source))+")", Dim)
		case *ast.AutoLink:
			out.add(SafeText(v.Label.Value(r.source)), style|Underline)
		case *ast.Image:
			out.add("[image: ", Dim)
			r.collectInline(v, style, depth+1, out)
			out.add("] ("+SafeText(v.Destination.Value(r.source))+")", Dim)
		case *ast.RawHTML:
			out.add(SafeText(v.Value.Value(r.source)), style)
		default:
			r.collectInline(node, style, depth+1, out)
		}
	}
}

// Keep a single text fragment by reference. Only adjacent fragments with the
// same style need a builder, so paragraph growth never recopies its full prefix.
type inlineText struct {
	spans  []Span
	first  string
	style  Style
	joined strings.Builder
}

func (s *inlineText) add(text string, style Style) {
	if text == "" {
		return
	}
	if s.first != "" && s.style != style {
		s.flush()
	}
	if s.first == "" {
		s.first, s.style = text, style
		return
	}
	if s.joined.Len() == 0 {
		s.joined.Grow(len(s.first) + len(text))
		s.joined.WriteString(s.first)
	}
	s.joined.WriteString(text)
}

func (s *inlineText) flush() {
	if s.first == "" {
		return
	}
	text := s.first
	if s.joined.Len() > 0 {
		text = s.joined.String()
	}
	s.spans = append(s.spans, Span{Text: text, Style: s.style})
	s.first = ""
	s.joined.Reset()
}
