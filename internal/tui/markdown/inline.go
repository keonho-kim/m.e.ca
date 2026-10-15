// Purpose: Convert inline Markdown nodes to safe monochrome text spans.
// Pattern: AST presentation projection.
// Usage: Block and table renderers call inline for their textual children.
// Related: internal/tui/markdown/render.go, internal/tui/markdown/layout.go.
package markdown

import (
	"github.com/yuin/goldmark/v2/ast"
)

func (r *renderer) inline(n ast.Node, style Style, depth int) []Span {
	if depth > 128 {
		r.failed = true
		return nil
	}
	var out []Span
	for node := n.FirstChild(); node != nil; node = node.NextSibling() {
		switch v := node.(type) {
		case *ast.Text:
			out = add(out, safe(v.Value.Value(r.source)), style)
			if v.HardLineBreak() {
				out = add(out, "\n", style)
			} else if v.SoftLineBreak() {
				out = add(out, " ", style)
			}
		case *ast.CodeSpan:
			out = add(out, safe(v.Value.Value(r.source)), style|Underline)
		case *ast.Strong:
			out = append(out, r.inline(v, style|Bold, depth+1)...)
		case *ast.Emphasis:
			out = append(out, r.inline(v, style|Italic, depth+1)...)
		case *ast.Link:
			out = append(out, r.inline(v, style|Underline, depth+1)...)
			out = add(out, " ("+safe(v.Destination.Value(r.source))+")", Dim)
		case *ast.AutoLink:
			out = add(out, safe(v.Label.Value(r.source)), style|Underline)
		case *ast.Image:
			out = add(out, "[image: ", Dim)
			out = append(out, r.inline(v, style, depth+1)...)
			out = add(out, "] ("+safe(v.Destination.Value(r.source))+")", Dim)
		case *ast.RawHTML:
			out = add(out, safe(v.Value.Value(r.source)), style)
		default:
			out = append(out, r.inline(node, style, depth+1)...)
		}
	}
	return out
}
