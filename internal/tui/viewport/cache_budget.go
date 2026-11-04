// Purpose: Bound aggregate derived display memory across history and active response blocks.
// Pattern: Event-loop-owned cache budget with compact completed rows and ordered eviction.
// Usage: Transcript trims after materializing visible rows; source and sparse anchors stay intact.
// Related: internal/tui/viewport/window.go, internal/tui/viewport/block_layout.go.
package viewport

import (
	"unsafe"

	"github.com/keonho-kim/meca/internal/tui/markdown"
)

const displayCacheBudget = 16 * 1024 * 1024

func (b *Block) cacheBytes() int {
	bytes := b.stream.RetainedBytes() + b.layout.parsedBytes + cap(b.layout.metrics)*int(unsafe.Sizeof(rowMetrics{}))
	if b.layout.packed != nil {
		bytes += b.layout.packed.RetainedBytes()
	}
	return bytes
}

// ensureMarkdown reports whether rows were recreated. Static line weights are
// accumulated by the index pass, or the visible-window recovery pass, exactly once.
func (b *Block) ensureMarkdown(width int) bool {
	l := &b.layout
	if l.parsed != nil || l.packed != nil {
		return false
	}
	if b.live {
		l.parsed = b.stream.Render(l.source, width)
	} else {
		b.stream = markdown.Stream{}
		if len(l.source) >= 128*1024 {
			l.packed = markdown.RenderPacked(l.source, width, displayCacheBudget)
			if l.packed != nil {
				return true
			}
		}
		l.parsed = markdown.Render(l.source, width)
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
	for _, block := range t.active {
		total += block.cacheBytes()
	}
	if total <= displayCacheBudget {
		return
	}
	// A completed representation can often shrink before any source needs to
	// be parsed again. Keep active incremental state under the same budget.
	for _, block := range t.visible {
		if total <= displayCacheBudget {
			return
		}
		if !block.live {
			before := block.cacheBytes()
			block.compactMarkdown()
			total += block.cacheBytes() - before
		}
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
	for _, block := range t.active {
		if total <= displayCacheBudget {
			return
		}
		total -= block.cacheBytes()
		block.releaseParsed()
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
	b.layout.packed = nil
	b.layout.metrics = nil
	b.layout.parsedBytes = 0
	b.stream = markdown.Stream{}
}
