// Purpose: Bound aggregate derived display memory across history and active response blocks.
// Pattern: Event-loop-owned cache budget with oldest-derived-first eviction.
// Usage: Transcript trims after materializing visible rows; source and sparse anchors stay intact.
// Related: internal/tui/viewport/window.go, internal/tui/viewport/block_layout.go.
package viewport

import (
	"unsafe"

	"github.com/keonho-kim/meca/internal/tui/markdown"
)

const displayCacheBudget = 16 * 1024 * 1024

func (b *Block) cacheBytes() int {
	return b.stream.RetainedBytes() + b.layout.parsedBytes + cap(b.layout.metrics)*int(unsafe.Sizeof(rowMetrics{}))
}

// ensureMarkdown reports whether rows were recreated. Static line weights are
// accumulated by the index pass, or the visible-window recovery pass, exactly once.
func (b *Block) ensureMarkdown(width int) bool {
	l := &b.layout
	if l.parsed != nil {
		return false
	}
	if b.live {
		l.parsed = b.stream.Render(b.Text, width)
	} else {
		b.stream = markdown.Stream{}
		l.parsed = markdown.Render(b.Text, width)
		l.parsedBytes = cap(l.parsed) * int(unsafe.Sizeof(markdown.Line{}))
	}
	return true
}

func (b *Block) retainParsedLine(line markdown.Line) {
	if !b.live {
		b.layout.parsedBytes += line.RetainedBytes()
	}
}

func (t *Transcript) enforceCacheBudget() {
	total := 0
	for _, block := range t.visible {
		if !block.live {
			total += block.cacheBytes()
		}
	}
	for _, block := range t.live {
		total += block.cacheBytes()
	}
	if total <= displayCacheBudget {
		return
	}
	// Finished history can be recreated from immutable source. Preserve active
	// incremental state whenever it fits; older Thinking yields before the reply.
	for _, block := range t.visible {
		if total <= displayCacheBudget {
			return
		}
		if !block.live {
			total -= block.cacheBytes()
			block.releaseParsed()
		}
	}
	for _, block := range t.live {
		if total <= displayCacheBudget {
			return
		}
		total -= block.cacheBytes()
		block.releaseParsed()
	}
}

// ReleaseDisplay retains scroll indexes but drops inactive parsing resources.
func (v *View) ReleaseDisplay() {
	for i := range v.Blocks {
		v.Blocks[i].visible = false
		v.Blocks[i].releaseParsed()
	}
}

func (b *Block) releaseParsed() {
	b.layout.parsed = nil
	b.layout.metrics = nil
	b.layout.parsedBytes = 0
	b.stream = markdown.Stream{}
}
