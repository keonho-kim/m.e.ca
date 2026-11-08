// Purpose: Distinguish original restoration budgets from visible text preview lengths.
// Pattern: Consumer-owned history read bounds.
// Usage: Server reads supply both bounds; terminal reads use their existing zero-value policy.
// Related: internal/storage/sqlite/timeline.go, internal/commands/server/snapshot.go.
package history

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

var ErrReadLimit = faults.Errorf(diagcatalog.HistoryStoredOriginalExceedsTheReadBudgetNarrow)

type ReadBounds struct {
	MaxBytes  int
	TextBytes int
}
