// Purpose: Distinguish original restoration budgets from visible text preview lengths.
// Pattern: Consumer-owned history read bounds.
// Usage: Server reads supply both bounds; terminal reads use their existing zero-value policy.
// Related: internal/storage/sqlite/timeline.go, internal/commands/server/snapshot.go.
package history

import "errors"

var ErrReadLimit = errors.New("stored original exceeds the read budget; narrow the query or export the session")

type ReadBounds struct {
	MaxBytes  int
	TextBytes int
}
