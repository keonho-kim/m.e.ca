// Purpose: Declare the persistence operations needed to reconcile interrupted work.
// Pattern: Consumer-owned repository contract.
// Usage: SQLite satisfies this contract; Restore consumes it.
// Related: internal/storage/recovery/recovery.go.
package recovery

import (
	"context"
	"github.com/keonho-kim/v-code/internal/agent/goal"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/agent/run"
)

type Store interface {
	LatestTask(context.Context, string) (*run.Checkpoint, error)
	LatestGoal(context.Context, string) (*goal.State, error)
	RecordEvent(context.Context, string, run.Event) error
	Messages(context.Context, string, int) ([]modelapi.Message, error)
	AppendMessage(context.Context, string, modelapi.Message) error
	LatestPlan(context.Context, string) (*planning.Plan, error)
	SavePlan(context.Context, planning.Plan) error
}
