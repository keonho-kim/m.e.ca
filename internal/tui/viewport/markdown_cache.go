// Purpose: Select completed compact rows or active parsed rows for visible Markdown wrapping.
// Pattern: Layout-owned representation adapter.
// Usage: Block.Window compacts oversized completed results before selecting visible rows.
// Related: internal/tui/viewport/block_window.go, internal/tui/viewport/cache_budget.go.
package viewport

import "github.com/keonho-kim/meca/internal/tui/markdown"

func (b *Block) compactMarkdown() {
	if b.live || b.layout.packed != nil || len(b.layout.parsed) == 0 {
		return
	}
	before := b.cacheBytes()
	if packed := markdown.Pack(b.layout.parsed, min(displayCacheBudget, before)); packed != nil && packed.RetainedBytes() < before {
		b.layout.packed = packed
		b.layout.parsed, b.layout.metrics = nil, nil
		b.layout.parsedBytes = 0
	}
}

func (l *blockLayout) markdownLines() int {
	if l.packed != nil {
		return l.packed.Len()
	}
	return len(l.parsed)
}
