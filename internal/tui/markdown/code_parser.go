// Purpose: Preserve native code-block parsing without retaining one source segment per line.
// Pattern: Goldmark parser adapter with parse-context-owned body buffers.
// Usage: documentParser registers these adapters before the native fenced and indented parsers.
// Related: internal/tui/markdown/parser.go, internal/tui/markdown/render.go.
package markdown

import (
	"strings"

	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/text"
)

var codeBodyKey = parser.NewContextKey()

type codeBody struct {
	value       strings.Builder
	scratch     []text.Segment
	lastContent int
}

type compactCodeParser struct{ native parser.BlockParser }

func (p compactCodeParser) Trigger() []byte             { return p.native.Trigger() }
func (p compactCodeParser) CanInterruptParagraph() bool { return p.native.CanInterruptParagraph() }
func (p compactCodeParser) CanAcceptIndentedLine() bool { return p.native.CanAcceptIndentedLine() }

func codeBodies(ctx parser.Context) map[ast.Node]*codeBody {
	values, _ := ctx.Get(codeBodyKey).(map[ast.Node]*codeBody)
	if values == nil {
		values = make(map[ast.Node]*codeBody)
		ctx.Set(codeBodyKey, values)
	}
	return values
}

func (p compactCodeParser) Open(parent ast.Node, reader text.Reader, ctx parser.Context) (ast.Node, parser.State) {
	node, state := p.native.Open(parent, reader, ctx)
	if node != nil {
		body := &codeBody{}
		codeBodies(ctx)[node] = body
		body.capture(node.(*ast.CodeBlock), reader.Source())
	}
	return node, state
}

func (p compactCodeParser) Continue(node ast.Node, reader text.Reader, ctx parser.Context) parser.State {
	body := codeBodies(ctx)[node]
	code := node.(*ast.CodeBlock)
	code.Value = text.NewLinesFromSegments(body.scratch[:0])
	state := p.native.Continue(node, reader, ctx)
	body.capture(code, reader.Source())
	return state
}

func (body *codeBody) capture(code *ast.CodeBlock, source []byte) {
	segments := code.Value.Segments()
	for _, segment := range segments {
		body.value.Write(segment.Bytes(source))
		if !segment.IsBlank(source) {
			body.lastContent = body.value.Len()
		}
	}
	body.scratch = segments[:0]
	code.Value = text.NewLinesFromString(body.value.String())
}

func (p compactCodeParser) Close(node ast.Node, reader text.Reader, ctx parser.Context) {
	values := codeBodies(ctx)
	body := values[node]
	code := node.(*ast.CodeBlock)
	code.Value = text.NewLinesFromSegments(body.scratch[:0])
	p.native.Close(node, reader, ctx)
	value := body.value.String()
	if code.CodeBlockKind == ast.CodeBlockKindIndented {
		value = value[:body.lastContent]
	}
	code.Value = text.NewLinesFromString(value)
	delete(values, node)
	if len(values) == 0 {
		ctx.Set(codeBodyKey, nil)
	}
}
