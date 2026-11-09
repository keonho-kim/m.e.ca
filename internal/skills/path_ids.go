// Purpose: Reuse bounded opaque path IDs without permanently consuming retired reader slots.
// Pattern: Reader-owned least-recently-used path registry.
// Usage: Path loads remember a canonical directory; ID reads refresh its use.
// Related: internal/skills/reader.go.
package skills

type pathEntry struct {
	path string
	used uint64
}

func (r *Reader) pathFor(id string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.paths[id]
	if !ok {
		return ""
	}
	r.pathClock++
	entry.used = r.pathClock
	r.paths[id] = entry
	return entry.path
}

func (r *Reader) rememberPath(id, path string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.paths == nil {
		r.paths = make(map[string]pathEntry)
	}
	if old, ok := r.paths[id]; ok {
		r.pathBytes -= len(old.path)
		delete(r.paths, id)
	}
	for len(r.paths) >= 128 || r.pathBytes+len(path) > 512<<10 {
		oldest := ""
		var age uint64
		for key, entry := range r.paths {
			if oldest == "" || entry.used < age {
				oldest, age = key, entry.used
			}
		}
		if oldest == "" {
			break
		}
		r.pathBytes -= len(r.paths[oldest].path)
		delete(r.paths, oldest)
	}
	r.pathClock++
	r.paths[id] = pathEntry{path: path, used: r.pathClock}
	r.pathBytes += len(path)
}
