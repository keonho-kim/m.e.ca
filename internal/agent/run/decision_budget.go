// Purpose: Stop one request when its model-call or time budget is exhausted.
// Pattern: Budget policy.
// Usage: The main loop checks limits before each decision and provider retry.
// Related: internal/agent/run/budget_usage.go, internal/agent/run/report.go.
package run

import (
	"context"
	"errors"
)

var errModelBudget = errors.New("model call budget exhausted")

func (s *execution) modelCall() bool { return s.usage.model(s.limits.MaxModelCalls) }

func (s *execution) prepareDecision(ctx context.Context) (bool, error) {
	if err := s.checkTimeBudget(ctx); err != nil {
		if ctx.Err() != nil {
			return true, ctx.Err()
		}
		return true, s.stopBudget(ctx, err.Error())
	}
	usage := s.usage.snapshot()
	if usage.ModelCalls >= s.limits.MaxModelCalls {
		return true, s.stopBudget(ctx, "model call budget exhausted")
	}
	return false, nil
}
