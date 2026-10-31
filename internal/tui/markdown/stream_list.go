// Purpose: Reuse completed top-level list items while preserving numbering and loose-list semantics.
// Pattern: Parser checkpoint with explicit rollback on structural changes.
// Usage: Stream retains a source boundary and row prefix, never an old AST.
// Related: internal/tui/markdown/stream.go, internal/tui/markdown/list.go.
package markdown

import "github.com/yuin/goldmark/v2/ast"

type listCheckpoint struct {
	at, before, weight, items, start int
	marker                           byte
	tight                            bool
}

type listProgress struct {
	node        *ast.List
	mutable     ast.Node
	before, row int
	items       int
}

func progressFor(document ast.Node) *listProgress {
	v, ok := document.LastChild().(*ast.List)
	if !ok || !v.IsTight || v.ChildCount() < 3 {
		return nil
	}
	mutable := v.LastChild().PreviousSibling()
	return &listProgress{node: v, mutable: mutable, items: v.ChildCount() - 2}
}

func (s *Stream) listDocument(source []byte) (ast.Node, bool) {
	document := documentParser.Parse(source)
	if s.list == nil {
		return document, true
	}
	v, ok := document.FirstChild().(*ast.List)
	if !ok || v.Marker != s.list.marker || v.IsTight != s.list.tight {
		return nil, false
	}
	// A source that writes every ordered marker as "1." still numbers items
	// sequentially. Loose lists use whole-list parsing because spacing is global.
	if v.IsOrdered() {
		v.Start = s.list.start + s.list.items
	}
	return document, true
}
