// Purpose: Reparse the mutable Markdown suffix while retaining completed top-level blocks.
// Pattern: Event-loop-owned incremental presentation cache.
// Usage: Viewport blocks own one Stream; replacing the block releases its parsed rows.
// Related: internal/tui/markdown/render.go, internal/tui/viewport/viewport.go.
package markdown

import (
	"strings"

	"github.com/yuin/goldmark/v2/ast"
)

type Stream struct {
	source         string
	cut, committed int
	rows           []Line
	result         []Line
	reused         int
}

// ReusedRows describes the immutable prefix retained by the most recent Render.
func (s *Stream) ReusedRows() int { return s.reused }

// Render accepts snapshots. The last two top-level blocks remain mutable for
// list continuation, tables and setext headings. References can affect earlier
// blocks, so a possible reference definition (]:) triggers complete parsing.
func (s *Stream) Render(text string) []Line {
	if s.source == text && s.result != nil {
		s.reused = len(s.result)
		return s.result
	}
	if text == "" {
		*s = Stream{}
		return nil
	}
	if len(text) > MaxParseBytes || strings.Contains(text, "]:") {
		*s = Stream{}
		return Render(text)
	}
	if s.cut > len(text) || s.cut > len(s.source) || text[:s.cut] != s.source[:s.cut] {
		*s = Stream{}
	}
	start := s.cut
	s.reused = s.committed
	r := renderer{source: []byte(text[start:])}
	if s.committed > 0 {
		r.rows = append(r.rows, s.rows[s.committed-1])
	}
	seed := len(r.rows)
	document := newParser().Parse(r.source)
	mutable := document.LastChild()
	if mutable != nil && mutable.PreviousSibling() != nil {
		mutable = mutable.PreviousSibling()
	}
	nextCut, nextRows := start, s.committed
	for node := document.FirstChild(); node != nil; node = node.NextSibling() {
		if node == mutable {
			if offset := blockStart(node, r.source); offset > 0 {
				nextCut, nextRows = start+offset, s.committed+len(r.rows)-seed
			}
		}
		r.block(node, "", 0)
	}
	if r.failed {
		*s = Stream{}
		return literal(text)
	}
	// Keep an untrimmed committed prefix: later blocks need its separator row.
	previous := s.rows
	s.rows = append(s.rows[:s.committed], r.rows[seed:]...)
	if len(previous) > len(s.rows) {
		clear(previous[len(s.rows):])
	}
	s.source, s.cut, s.committed = text, nextCut, nextRows
	end := len(s.rows)
	for end > 0 && s.rows[end-1].Text == "" {
		end--
	}
	count := len(text) - len(strings.TrimRight(text, "\n"))
	if end == 0 && count == 0 {
		s.result = nil
		return nil
	}
	// Only trailing blank rows are replaced; the committed separator precedes
	// the two mutable blocks and is never overwritten by this normalization.
	s.result = append(s.rows[:end], make([]Line, count)...)
	return s.result
}

func blockStart(node ast.Node, source []byte) int {
	at := node.Pos()
	if at < 0 || at > len(source) {
		return 0
	}
	return strings.LastIndexByte(string(source[:at]), '\n') + 1
}
