// Purpose: Enforce total execution time only before operations without interrupting active model reception.
// Pattern: Request budget policy.
// Usage: Model-call boundaries check the request deadline; tools use it as their deadline.
// Related: internal/agent/run/run.go, internal/agent/run/usage.go, internal/agent/run/execute.go.
package run

import (
	"context"

	"time"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

var errTimeBudget = faults.Errorf(diagcatalog.RunExecutionTimeBudgetExhaustedBeforeTheNext)

func (s *execution) checkTimeBudget(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !s.usage.deadline.IsZero() && !time.Now().Before(s.usage.deadline) {
		return errTimeBudget
	}
	return nil
}
