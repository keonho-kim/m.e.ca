// Purpose: Stop one request when its model-call or time budget is exhausted.
// Pattern: Budget policy.
// Usage: The main loop checks limits before each decision and provider retry.
// Related: internal/agent/run/budget_usage.go, internal/agent/run/report.go.
package run

import (
	"context"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

var errModelBudget = faults.Errorf(diagcatalog.RunModelCallBudgetExhausted)

func (s *execution) modelCall() bool {
	if !s.usage.model(s.limits.MaxModelCalls) {
		return false
	}
	return s.runner.AdmitModel == nil || s.runner.AdmitModel()
}

func (s *execution) prepareDecision(ctx context.Context) (bool, error) {
	if s.result.Departed {
		s.checkpoint.Status = "answered"
		return true, nil
	}
	if err := s.checkTimeBudget(ctx); err != nil {
		if ctx.Err() != nil {
			return true, ctx.Err()
		}
		return true, s.stopBudget(ctx, err.Error())
	}
	if s.usage.ModelCalls >= s.limits.MaxModelCalls {
		return true, s.stopBudget(ctx, "model call budget exhausted")
	}
	return false, nil
}
