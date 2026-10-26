// Purpose: Declare the persistence operations needed to reconcile interrupted work.
// Pattern: Consumer-owned repository contract.
// Usage: SQLite satisfies this contract; Restore consumes it.
// Related: internal/storage/recovery/recovery.go.
package recovery

import (
	"context"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

type Store interface {
	LatestRun(context.Context, string) (*runrecord.Checkpoint, error)
	RecordEvent(context.Context, string, runrecord.Event) error
	Messages(context.Context, string, int) ([]modelapi.Message, error)
	Commit(context.Context, string, []runrecord.Entry) error
}
