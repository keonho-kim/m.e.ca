// Purpose: Build sparse row positions and edge metrics for literal transcript text.
// Pattern: Pure index traversal over source-bound rows.
// Usage: Block.prepare delegates non-Markdown text to indexLiteral.
// Related: internal/tui/viewport/block_layout.go, internal/tui/textlayout/wrap.go.
package viewport

import (
	"strings"

	"github.com/keonho-kim/meca/internal/tui/textlayout"
)

func (b *Block) indexLiteral(width int) (firstContent, lastContent int) {
	l := &b.layout
	firstContent, lastContent = -1, -1
	for row := range textlayout.Rows(l.source, width) {
		if l.total%256 == 0 {
			l.points = append(l.points, blockPoint{source: row.Start, row: l.total, prefix: row.Prefix})
		}
		if strings.TrimSpace(row.Text) != "" {
			if firstContent < 0 {
				firstContent = l.total
			}
			lastContent = l.total
		}
		l.total++
	}
	return firstContent, lastContent
}
