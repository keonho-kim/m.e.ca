// Purpose: Render large independent top-level Markdown groups with bounded temporary workers.
// Pattern: Ordered fork/join over one immutable parsed document.
// Usage: Full rendering joins all groups before publishing rows; live suffixes stay incremental.
// Related: internal/tui/markdown/render.go, internal/tui/markdown/stream.go, internal/tui/markdown/render_packed.go.
package markdown

import (
	"sync"

	"github.com/yuin/goldmark/v2/ast"
	extast "github.com/yuin/goldmark/v2/extension/ast"
)

func (r *renderer) document(parent ast.Node) {
	starts, parallel := partitionDocument(parent, len(r.source))
	if !parallel {
		r.blocks(parent, "", 0)
		return
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

// partitionDocument shares measured fork/join boundaries across row representations.
func partitionDocument(parent ast.Node, sourceBytes int) ([5]ast.Node, bool) {
	count := parent.ChildCount()
	if sourceBytes < 128*1024 || count < 32 {
		return [5]ast.Node{}, false
	}
	for n := parent.FirstChild(); n != nil; n = n.NextSibling() {
		if _, table := n.(*extast.Table); table {
			// Concurrent table layout raises peak memory through cell/span
			// scratch allocations. Keep the measured low-memory serial path.
			return [5]ast.Node{}, false
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
	return starts, true
}
