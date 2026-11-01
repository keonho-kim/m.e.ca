// Purpose: Render large independent top-level Markdown groups with bounded temporary workers.
// Pattern: Ordered fork/join over one immutable parsed document.
// Usage: Full rendering joins all groups before publishing rows; live suffixes stay incremental.
// Related: internal/tui/markdown/render.go, internal/tui/markdown/stream.go.
package markdown

import (
	"sync"

	"github.com/yuin/goldmark/v2/ast"
	extast "github.com/yuin/goldmark/v2/extension/ast"
)

func (r *renderer) document(parent ast.Node) {
	count := parent.ChildCount()
	if len(r.source) < 128*1024 || count < 32 {
		r.blocks(parent, "", 0)
		return
	}
	for n := parent.FirstChild(); n != nil; n = n.NextSibling() {
		if _, table := n.(*extast.Table); table {
			// Concurrent table layout raises peak memory through cell/span
			// scratch allocations. Keep the measured low-memory serial path.
			r.blocks(parent, "", 0)
			return
		}
	}
	var starts [5]ast.Node
	starts[0] = parent.FirstChild()
	width := (count + 3) / 4
	index := 0
	for n := parent.FirstChild(); n != nil; n = n.NextSibling() {
		if index > 0 && index%width == 0 {
			starts[index/width] = n
		}
		index++
	}
	var groups [4]renderer
	var workers sync.WaitGroup
	for i := range groups {
		workers.Go(func() {
			part := &groups[i]
			part.source, part.width = r.source, r.width
			for n := starts[i]; n != nil && n != starts[i+1]; n = n.NextSibling() {
				part.block(n, "", 0)
			}
		})
	}
	workers.Wait()
	rows := len(groups)
	for i := range groups {
		rows += len(groups[i].rows)
	}
	r.rows = make([]Line, 0, rows)
	for i := range groups {
		part := &groups[i]
		if part.leadingBlank {
			r.blank()
		}
		r.rows = append(r.rows, part.rows...)
		r.failed = r.failed || part.failed
		part.rows = nil
	}
}
