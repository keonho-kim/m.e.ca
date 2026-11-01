// Purpose: Separate contiguous workflow output with full-width horizontal rules.
// Pattern: Pure window decoration over the existing sparse transcript index.
// Usage: Transcript inserts rules around labelled groups without copying their bodies.
// Related: internal/tui/viewport/window.go, internal/tui/viewport/transcript_index.go.
package viewport

import (
	"strings"

	"github.com/keonho-kim/meca/internal/tui/textlayout"
)

// Group identifies consecutive blocks sharing one workflow title.
type Group struct {
	ID, Title string
	Open      bool
}

func (p section) origin() int {
	if p.edge == '+' {
		return p.block.layout.trim - 1
	}
	if p.edge == '-' {
		return p.block.layout.total
	}
	return p.block.layout.trim + p.skip
}

func (p section) divider(width int) Line {
	text := strings.Repeat("-", max(0, width))
	if title := " " + p.group.Title + " "; p.edge == '+' && textlayout.Width(title) <= width {
		left := (width - textlayout.Width(title)) / 2
		text = text[:left] + title + text[left+textlayout.Width(title):]
	}
	return Line{Text: text, ID: p.block.ID, Row: p.origin(), Thinking: true}
}
