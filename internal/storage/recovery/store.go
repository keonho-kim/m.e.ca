// Purpose: Declare the persistence operations needed to reconcile interrupted work.
// Pattern: Consumer-owned repository contract.
// Usage: SQLite satisfies this contract; Restore consumes it.
// Related: internal/storage/recovery/recovery.go.
package recovery

import (
	"context"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type Store interface {
	LatestRun(context.Context, string) (*runrecord.Checkpoint, error)
	RecordEvent(context.Context, string, runrecord.Event) error
	PendingTools(context.Context, string) ([]toolapi.Call, error)
	Commit(context.Context, string, []runrecord.Entry) error
}
