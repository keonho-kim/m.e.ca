// Purpose: Reuse completion metadata and retain independent bounded pagination snapshots.
// Pattern: Reader-owned immutable cache with cancellable scan admission.
// Usage: List and completion reuse metadata; document reads always reopen and revalidate files.
// Related: internal/skills/catalog.go, internal/skills/reader.go, internal/skills/suggestions.go.
package skills

import (
	"context"
	"crypto/rand"
	"os"
	"slices"
	"time"

	"github.com/keonho-kim/meca/internal/skills/instruction"
)

type catalogSnapshot struct {
	id, scope   string
	items       []catalogEntry
	diagnostics []Diagnostic
	truncated   bool
	version     int
	at          time.Time
}

// Cached sizing belongs to the reader snapshot, not the instruction value
// exchanged with tools, model context or callers.
type catalogEntry struct {
	instruction.Metadata
	bytes int
}

func newCatalogEntry(metadata instruction.Metadata) catalogEntry {
	return catalogEntry{Metadata: metadata, bytes: metadataBytes(metadata)}
}

func catalogScope(ctx context.Context) string {
	if root, ok := ctx.Value(readRootKey{}).(*os.Root); ok && root != nil {
		return root.Name()
	}
	return ""
}

func (r *Reader) snapshot(ctx context.Context, fresh bool) (*catalogSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	scope := catalogScope(ctx)
	current := func() *catalogSnapshot {
		r.mu.RLock()
		defer r.mu.RUnlock()
		cached := r.cached
		if cached != nil && cached.scope == scope && cached.version == r.version() {
			age := r.now().Sub(cached.at)
			if age >= 0 && age < time.Second {
				return cached
			}
		}
		return nil
	}
	if !fresh {
		if cached := current(); cached != nil {
			return cached, nil
		}
	}
	select {
	case r.catalogGate <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-r.catalogGate }()
	if !fresh {
		if cached := current(); cached != nil {
			return cached, nil
		}
	}
	for range 2 {
		version := r.version()
		items, diagnostics, truncated, err := r.catalog(ctx)
		if err != nil {
			return nil, err
		}
		if version != r.version() {
			continue
		}
		cached := &catalogSnapshot{id: rand.Text(), scope: scope, items: items, diagnostics: diagnostics, truncated: truncated, version: version, at: r.now()}
		return r.retainCatalog(cached)
	}
	return nil, ErrInvalid
}

func (r *Reader) pageSnapshot(ctx context.Context, id string) (*catalogSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	cached := r.snapshots[id]
	if cached == nil || cached.scope != catalogScope(ctx) || cached.version != r.version() {
		return nil, ErrInvalid
	}
	if age := r.now().Sub(cached.at); age < 0 || age > catalogLifetime {
		return nil, ErrInvalid
	}
	return cached, nil
}

func cloneMetadata(value instruction.Metadata) instruction.Metadata {
	value.Warnings = slices.Clone(value.Warnings)
	return value
}
