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

func (s *execution) modelCall(ctx context.Context) error {
	if s.usage.ModelCalls >= s.limits.MaxModelCalls {
		return errModelBudget
	}
	if s.runner.AdmitModel != nil {
		admitted, err := s.runner.AdmitModel(ctx)
		if err != nil {
			return err
		}
		if !admitted {
			return errModelBudget
		}
	}
	s.usage.ModelCalls++
	return nil
}

func (s *execution) prepareDecision(ctx context.Context, hasDraft bool) (bool, error) {
	if s.result.Departed {
		s.checkpoint.Status = "answered"
		return true, nil
	}
	if err := s.checkTimeBudget(ctx); err != nil {
		if ctx.Err() != nil || hasDraft {
			return true, err
		}
		return true, s.stopBudget(ctx, err.Error())
	}
	if s.usage.ModelCalls >= s.limits.MaxModelCalls {
		if hasDraft {
			return true, errModelBudget
		}
		return true, s.stopBudget(ctx, "model call budget exhausted")
	}
	return false, nil
}
