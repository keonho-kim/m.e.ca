// Purpose: Keep independent bounded catalog traversals alive while completion metadata refreshes.
// Pattern: Reader-owned immutable snapshot retention.
// Usage: A completed scan is retained for pagination; expiry and capacity evict old traversals.
// Related: internal/skills/snapshot.go, internal/skills/catalog.go.
package skills

import (
	"slices"
	"time"
)

const catalogLifetime = 2 * time.Minute
const catalogBytes = 8 << 20
const catalogSnapshots = 8

func sameCatalog(a, b *catalogSnapshot) bool {
	return a.scope == b.scope && a.version == b.version && a.truncated == b.truncated && slices.Equal(a.diagnostics, b.diagnostics) && slices.EqualFunc(a.items, b.items, func(x, y catalogEntry) bool {
		return x.ID == y.ID && x.Name == y.Name && x.Description == y.Description && x.Source == y.Source && x.ManualOnly == y.ManualOnly && slices.Equal(x.Warnings, y.Warnings)
	})
}

func (s *catalogSnapshot) bytes() int {
	size := 256
	for _, m := range s.items {
		size += m.bytes
	}
	for _, d := range s.diagnostics {
		size += 64 + len(d.ID) + len(d.Message)
	}
	return size
}

func (r *Reader) retainCatalog(next *catalogSnapshot) (*catalogSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.snapshots == nil {
		r.snapshots = make(map[string]*catalogSnapshot)
	}
	cost := next.bytes()
	if cost > catalogBytes {
		return nil, ErrLimit
	}
	used := 0
	var matching *catalogSnapshot
	for id, old := range r.snapshots {
		age := next.at.Sub(old.at)
		if age < 0 || age > catalogLifetime || old.version != next.version {
			delete(r.snapshots, id)
			continue
		}
		if sameCatalog(old, next) {
			matching = old
		}
		used += old.bytes()
	}
	if matching != nil {
		matching.at = next.at
		r.cached = matching
		return matching, nil
	}
	for len(r.snapshots) >= catalogSnapshots || used+cost > catalogBytes && len(r.snapshots) > 0 {
		oldest := ""
		var at time.Time
		for id, old := range r.snapshots {
			if oldest == "" || old.at.Before(at) {
				oldest, at = id, old.at
			}
		}
		used -= r.snapshots[oldest].bytes()
		delete(r.snapshots, oldest)
	}
	r.snapshots[next.id] = next
	r.cached = next
	return next, nil
}
