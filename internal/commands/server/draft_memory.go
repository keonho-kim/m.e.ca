// Purpose: Bound retained draft text and previews across server requests.
// Pattern: Nonblocking process byte ledger and a small snapshot prefix.
// Usage: Draft originals spill to disk when memory is unavailable; previews can be paged on demand.
// Related: internal/commands/server/draft_spool.go, internal/commands/server/stream_projection.go.
package server

import (
	"strings"
	"sync"
)

const (
	draftMemoryBytes  = 16 << 20
	draftPreviewBytes = 4 << 10
)

var draftMemory struct {
	sync.Mutex
	used int
}

func reserveDraftMemory(bytes int) bool {
	draftMemory.Lock()
	defer draftMemory.Unlock()
	if bytes > draftMemoryBytes-draftMemory.used {
		return false
	}
	draftMemory.used += bytes
	return true
}

func releaseDraftMemory(bytes int) {
	if bytes == 0 {
		return
	}
	draftMemory.Lock()
	draftMemory.used -= bytes
	draftMemory.Unlock()
}

type draftPreview struct {
	body strings.Builder
}

func (p *draftPreview) update(snapshot string) string {
	if snapshot == "" {
		return ""
	}
	if p.body.Cap() == 0 {
		if !reserveDraftMemory(draftPreviewBytes) {
			return ""
		}
		p.body.Grow(draftPreviewBytes)
	}
	if remaining := draftPreviewBytes - p.body.Len(); remaining > 0 {
		p.body.WriteString(prefix(snapshot[p.body.Len():], remaining))
	}
	return p.body.String()
}

func (p *draftPreview) clear() {
	if p.body.Cap() > 0 {
		p.body.Reset()
		releaseDraftMemory(draftPreviewBytes)
	}
}
