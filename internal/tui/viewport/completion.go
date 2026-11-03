// Purpose: Transfer matching streamed Markdown rows to bounded completed display ownership.
// Pattern: Event-loop-owned lifecycle transition.
// Usage: Update and FinishStream end live state without reparsing unchanged source.
// Related: internal/tui/viewport/cache_budget.go, internal/tui/viewport/markdown_cache.go.
package viewport

import (
	"unsafe"

	"github.com/keonho-kim/meca/internal/tui/markdown"
)

func (b *Block) finishStream() {
	b.Streaming, b.live = false, false
	l := &b.layout
	if !b.visible || !l.ready || l.key != (blockKey{b.Text, l.key.width, b.ID, b.Markdown, b.Thinking, b.Centered, b.Changes}) {
		b.releaseParsed()
		return
	}
	// The stream and layout share rows. Keep the immutable result, dropping
	// mutable checkpoints and per-line metrics before giving it a final weight.
	b.stream = markdown.Stream{}
	l.metrics = nil
	l.parsedBytes = cap(l.parsed) * int(unsafe.Sizeof(markdown.Line{}))
	for _, row := range l.parsed {
		l.parsedBytes += row.RetainedBytes()
	}
	if b.cacheBytes() > displayCacheBudget {
		b.compactMarkdown()
		if b.cacheBytes() > displayCacheBudget {
			b.releaseParsed()
		}
	}
}
