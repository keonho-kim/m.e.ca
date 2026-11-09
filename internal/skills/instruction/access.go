// Purpose: Carry explicit user read intent through a request without granting filesystem permissions.
// Pattern: Immutable context capability.
// Usage: Runtime mentions and human server previews authorize manual-only instruction reads.
// Related: internal/skills/reader.go, internal/skills/assets.go, internal/agent/run/skills.go.
package instruction

import (
	"context"
	"slices"
)

type explicitReadKey struct{}

func WithExplicitReads(ctx context.Context, ids []string) context.Context {
	return context.WithValue(ctx, explicitReadKey{}, slices.Clone(ids))
}

func ExplicitRead(ctx context.Context, id string) bool {
	ids, _ := ctx.Value(explicitReadKey{}).([]string)
	for _, allowed := range ids {
		if allowed == id {
			return true
		}
	}
	return false
}
