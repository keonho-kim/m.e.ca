// Purpose: Bind project-scoped path reads to an already-owned filesystem root.
// Pattern: I/O access boundary carried by the execution context.
// Usage: Local tools attach their pinned project root for directory-scoped arbitrary paths.
// Related: internal/skills/files.go, internal/tools/local/skills.go.
package skills

import (
	"context"
	"os"
)

type readRootKey struct{}

func WithReadRoot(ctx context.Context, root *os.Root) context.Context {
	return context.WithValue(ctx, readRootKey{}, root)
}
