// Purpose: Enforce total task time only before operations without interrupting active model reception.
// Pattern: Task budget policy.
// Usage: Model-call boundaries check the shared worker deadline; tools use it as their deadline.
// Related: internal/agent/run/run.go, internal/agent/run/usage.go, internal/agent/run/execute.go.
package run

import (
	"context"
	"fmt"
	"time"
)

func (s *task) checkTimeBudget(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !s.usage.deadline.IsZero() && !time.Now().Before(s.usage.deadline) {
		return fmt.Errorf("task time budget exhausted before the next operation")
	}
	return nil
}
