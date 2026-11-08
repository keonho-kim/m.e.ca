// Purpose: Select bounded name-prefix completions from metadata in source-priority order.
// Pattern: Pure selection over on-demand discovery.
// Usage: The terminal completion worker calls Suggestions without loading instruction bodies.
// Related: internal/skills/catalog.go, internal/tui/session/skills_hints.go.
package skills

import (
	"context"
	"strings"

	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func (r *Reader) Suggestions(ctx context.Context, prefix string) ([]instruction.Metadata, error) {
	if len(prefix) > 512 {
		return nil, ErrInvalid
	}
	snapshot, err := r.snapshot(ctx, false)
	if err != nil {
		return nil, err
	}
	var result []instruction.Metadata
	seen, bytes := make(map[string]bool), 0
	for _, m := range snapshot.items {
		name := m.Name
		if strings.Contains(prefix, ":") {
			name = m.ID
		}
		if seen[name] || !strings.HasPrefix(name, prefix) {
			continue
		}
		size := m.bytes
		if len(result) >= 32 || bytes+size > MaxPageBytes {
			break
		}
		seen[name], bytes = true, bytes+size
		result = append(result, cloneMetadata(m.Metadata))
	}
	return result, nil
}
