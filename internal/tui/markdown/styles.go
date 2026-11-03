// Purpose: Read style boundaries from ordinary spans or compact completed rows.
// Pattern: Monotonic value cursor over immutable style runs.
// Usage: Shared wrapping calls at while moving forward through one logical line.
// Related: internal/tui/markdown/layout.go, internal/tui/markdown/packed.go.
package markdown

type spanCursor struct {
	spans            []Span
	packed           []packedSpan
	index, end, base int
}

func (c *spanCursor) at(position int) (Style, int) {
	if c.packed != nil {
		for c.index+1 < len(c.packed) && int(c.packed[c.index].end)-c.base <= position {
			c.index++
		}
		run := c.packed[c.index]
		return run.style, int(run.end) - c.base
	}
	for c.index+1 < len(c.spans) && c.end <= position {
		c.index++
		c.end += len(c.spans[c.index].Text)
	}
	return c.spans[c.index].Style, c.end
}
