// Purpose: Emit Markdown rows incrementally while preserving list markers and trailing blank semantics.
// Pattern: Renderer-owned row sink with bounded pending whitespace.
// Usage: Packed rendering and Walk consume rows directly; Render retains its ordinary row slice.
// Related: internal/tui/markdown/render.go, internal/tui/markdown/list.go, internal/tui/markdown/walk.go.
package markdown

import "slices"

type rowSink struct {
	send           func(Line) bool
	total, pending int
}

func (r *renderer) rowCount() int {
	if r.sink != nil {
		return r.sink.total
	}
	return len(r.rows)
}

func (r *renderer) lastBlank() bool {
	if r.sink != nil {
		return r.sink.pending > 0
	}
	return len(r.rows) > 0 && r.rows[len(r.rows)-1].Text == ""
}

func (r *renderer) appendRow(value Line) {
	if r.stopped || r.failed {
		return
	}
	if transform := r.firstRow; transform != nil {
		r.firstRow = nil
		value = transform(value)
	}
	if r.sink == nil {
		r.rows = append(r.rows, value)
		return
	}
	s := r.sink
	s.total++
	if value.Text == "" {
		s.pending++
		return
	}
	for s.pending > 0 {
		s.pending--
		if !s.send(Line{}) {
			r.stopped = true
			return
		}
	}
	r.stopped = !s.send(value)
}

func (r *renderer) trimBlankRows(start int) {
	if r.sink != nil {
		s := r.sink
		removed := min(s.pending, s.total-start)
		s.pending, s.total = s.pending-removed, s.total-removed
		return
	}
	for len(r.rows) > start && r.lastBlank() {
		r.rows[len(r.rows)-1] = Line{}
		r.rows = r.rows[:len(r.rows)-1]
	}
}

func listMarker(value Line, continuation, marker string) Line {
	spans := slices.Clone(value.Spans)
	if len(spans) > 0 && len(spans[0].Text) >= len(continuation) && spans[0].Text[:len(continuation)] == continuation {
		spans[0].Text = spans[0].Text[len(continuation):]
	}
	return line(append([]Span{{Text: marker, Style: Dim}}, spans...))
}
