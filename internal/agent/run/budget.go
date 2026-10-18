// Purpose: Enforce total execution time only before operations without interrupting active model reception.
// Pattern: Task budget policy.
// Usage: Model-call boundaries check the request deadline; tools use it as their deadline.
// Related: internal/agent/run/run.go, internal/agent/run/usage.go, internal/agent/run/execute.go.
package run

import (
	"context"
	"errors"
	"time"
)

var errTimeBudget = errors.New("execution time budget exhausted before the next operation")

func (s *execution) checkTimeBudget(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !s.usage.snapshot().deadline.IsZero() && !time.Now().Before(s.usage.snapshot().deadline) {
		return errTimeBudget
	}
	return nil
}
