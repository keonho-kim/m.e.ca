// Purpose: Reparse the mutable Markdown suffix while retaining completed top-level blocks.
// Pattern: Event-loop-owned incremental presentation cache.
// Usage: Viewport blocks own one Stream; replacing the block releases its parsed rows.
// Related: internal/tui/markdown/render.go, internal/tui/viewport/viewport.go.
package markdown

import (
	"strings"
	"unsafe"

	"github.com/yuin/goldmark/v2/ast"
)

type Stream struct {
	source                  string
	cut, committed          int
	rows                    []Line
	result                  []Line
	reused                  int
	width                   int
	weight, committedWeight int
	list                    *listCheckpoint
}

// ReusedRows describes the immutable prefix retained by the most recent Render.
func (s *Stream) ReusedRows() int { return s.reused }

// RetainedBytes conservatively counts backing arrays and derived text. Aliased
// strings can be counted twice; this is a cache eviction weight, not a heap metric.
func (s *Stream) RetainedBytes() int {
	return s.weight + (cap(s.rows)+cap(s.result))*int(unsafe.Sizeof(Line{}))
}

func lineWeight(line Line) int {
	weight := len(line.Text) + cap(line.Spans)*int(unsafe.Sizeof(Span{}))
	for _, span := range line.Spans {
		weight += len(span.Text)
	}
	return weight
}

// Render accepts snapshots. The last two top-level blocks remain mutable for
// list continuation, tables and setext headings. References can affect earlier
// blocks, so a possible reference definition (]:) triggers complete parsing.
// A width change invalidates retained table layouts as well as the mutable tail.
func (s *Stream) Render(text string, width int) []Line {
	if s.width != width {
		*s = Stream{width: width}
	}
	if s.source == text && s.result != nil {
		s.reused = len(s.result)
		return s.result
	}
	if text == "" || width < 1 {
		*s = Stream{}
		return nil
	}
	if len(text) > MaxParseBytes || strings.Contains(text, "]:") {
		return s.whole(text, width)
	}
	if s.cut > len(text) || s.cut > len(s.source) || text[:s.cut] != s.source[:s.cut] {
		*s = Stream{width: width}
	}
	source := []byte(text[s.cut:])
	document, valid := s.listDocument(source)
	if !valid {
		p := s.list
		s.cut, s.committed, s.committedWeight, s.list = p.at, p.before, p.weight, nil
		source = []byte(text[s.cut:])
		document, _ = s.listDocument(source)
	}
	start := s.cut
	s.reused = s.committed
	r := renderer{source: source, width: width}
	if s.committed > 0 {
		r.rows = append(r.rows, s.rows[s.committed-1])
	}
	seed := len(r.rows)
	r.listProgress = progressFor(document)
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
		return s.whole(text, width)
	}
	var checkpoint *listCheckpoint
	if s.list != nil && nextCut == start {
		checkpoint = s.list
	}
	if p := r.listProgress; p != nil {
		checkpoint = &listCheckpoint{at: start + blockStart(p.node, r.source), before: s.committed + p.before - seed, marker: p.node.Marker, tight: p.node.IsTight, start: p.node.Start, items: p.items}
		if s.list != nil && document.FirstChild() == p.node {
			checkpoint.at, checkpoint.before, checkpoint.weight = s.list.at, s.list.before, s.list.weight
			checkpoint.start, checkpoint.items = s.list.start, s.list.items+p.items
		}
		nextCut, nextRows = start+blockStart(p.mutable, r.source), s.committed+p.row-seed
	}
	weight, nextWeight := s.committedWeight, s.committedWeight
	for i, row := range r.rows[seed:] {
		if checkpoint != nil && s.committed+i == checkpoint.before {
			checkpoint.weight = weight
		}
		weight += lineWeight(row)
		if s.committed+i+1 == nextRows {
			nextWeight = weight
		}
	}
	s.weight, s.committedWeight, s.list = weight, nextWeight, checkpoint
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

func (s *Stream) whole(text string, width int) []Line {
	*s = Stream{source: text, width: width, result: Render(text, width)}
	for _, row := range s.result {
		s.weight += lineWeight(row)
	}
	return s.result
}

func blockStart(node ast.Node, source []byte) int {
	at := node.Pos()
	if at < 0 || at > len(source) {
		return 0
	}
	return strings.LastIndexByte(string(source[:at]), '\n') + 1
}
