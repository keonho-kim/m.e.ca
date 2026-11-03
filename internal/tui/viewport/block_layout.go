// Purpose: Coordinate source-bound block indexing and presentation boundary trimming.
// Pattern: Event-loop-owned presentation index.
// Usage: Transcript queries Count and Window; block replacement releases its index.
// Related: internal/tui/viewport/window.go, internal/tui/markdown/stream.go, internal/tui/textlayout/wrap.go.
package viewport

import (
	"strings"

	"github.com/keonho-kim/meca/internal/tui/markdown"
)

const maxBoundaryBlanks = 2

type blockPoint struct {
	source, row    int
	offset, within int
	prefix         string
	last           int
}
type blockLayout struct {
	key                            blockKey
	ready                          bool
	points                         []blockPoint
	parsed                         []markdown.Line
	packed                         *markdown.Packed
	metrics                        []rowMetrics
	source                         string
	markdown                       bool
	total, trim, leading, trailing int
	first                          int
	parsedBytes                    int
}

type rowMetrics struct{ count, leading, trailing int }

func (b *Block) prepare(width int) {
	key := blockKey{b.Text, width, b.ID, b.Markdown, b.Thinking, b.Centered, b.SpaceAfter, b.Changes}
	l := &b.layout
	if l.ready && l.key == key {
		return
	}
	previous := *l
	*l = blockLayout{key: key, ready: true}
	l.source = b.Text
	if b.Markdown || b.Thinking || b.SpaceAfter > 0 {
		// Chat displays at most two boundary blanks. Keep the original block
		// intact, but never allocate Markdown rows for blanks that will be hidden.
		end := len(strings.TrimRight(l.source, "\n"))
		l.source = l.source[:min(len(l.source), end+maxBoundaryBlanks)]
	}
	l.markdown = b.Markdown && len(l.source) <= markdown.MaxParseBytes
	if b.Markdown && !l.markdown {
		l.source = markdown.SafeText(l.source)
	}
	if width < 1 || strings.TrimSpace(b.Text) == "" {
		b.stream = markdown.Stream{}
		return
	}
	if l.markdown {
		b.ensureMarkdown(width)
	}
	var firstContent, lastContent int
	if l.markdown {
		firstContent, lastContent = b.indexMarkdown(previous, width)
	} else {
		b.stream = markdown.Stream{}
		firstContent, lastContent = b.indexLiteral(width)
	}
	l.first = firstContent
	if firstContent < 0 {
		// Reference definitions can parse successfully without producing any
		// visible content. Display spacing must not manufacture source rows.
		l.total = 0
		return
	}
	if b.Markdown || b.Thinking || b.SpaceAfter > 0 {
		l.trim = max(0, firstContent)
		l.total = min(l.total, lastContent+1+maxBoundaryBlanks)
	}
	l.leading = max(0, firstContent-l.trim)
	l.trailing = l.total - lastContent - 1
	if b.SpaceAfter > 0 {
		// Spacing is display metadata: never copy a streamed body just to append
		// newlines, and replace source boundary blanks rather than adding to them.
		l.trailing = min(int(b.SpaceAfter), maxBoundaryBlanks)
		l.total = lastContent + 1 + l.trailing
	}
}

func (b *Block) Count(width int) int { b.prepare(width); return b.layout.total - b.layout.trim }
