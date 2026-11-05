// Purpose: Batch independent peer commands while keeping cooperation and file preparation boundaries serial.
// Pattern: Team-scoped scheduling adapter to the existing local command planner.
// Usage: The ordinary run loop discovers ParallelBatch on the team executor.
// Related: internal/tools/team/executor.go, internal/tools/local/parallel.go, internal/agent/run/batch.go.
package team

import (
	"context"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

func (e *Executor) ParallelBatch(ctx context.Context, settings policy.Settings, calls []toolapi.Call, maximum int) (toolapi.BatchPlan, error) {
	if len(calls) == 0 {
		return toolapi.BatchPlan{PreparationConcurrency: 1}, nil
	}
	if calls[0].Name != "execute_command" {
		return toolapi.BatchPlan{Count: 1, PreparationConcurrency: 1}, nil
	}
	end := 0
	for end < len(calls) && calls[end].Name == "execute_command" {
		end++
	}
	return e.Local.ParallelBatch(ctx, settings, calls[:end], maximum)
}
