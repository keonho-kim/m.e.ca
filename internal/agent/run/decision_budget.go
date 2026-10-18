// Purpose: Stop one request when its action, model or time budget is exhausted.
// Pattern: Budget policy.
// Usage: The main loop checks limits before each decision and provider retry.
// Related: internal/agent/run/budget_usage.go, internal/agent/run/report.go.
package run

import (
	"context"
	"errors"
	"github.com/keonho-kim/v-code/internal/agent/work"
)

var errModelBudget = errors.New("model call budget exhausted")

func (s *execution) modelCall() bool { return s.usage.model(s.limits.MaxModelCalls) }

func (s *execution) prepareDecision(ctx context.Context) (bool, error) {
	if err := s.refreshFileFacts(ctx); err != nil {
		return true, err
	}
	if end := s.checkpoint.End; end != nil && end.Status == work.Denied {
		_, err := s.report(ctx, end.Reason, true)
		return true, err
	}
	if err := s.checkTimeBudget(ctx); err != nil {
		if ctx.Err() != nil {
			return true, ctx.Err()
		}
		_, err := s.budgetReport(ctx, err.Error())
		return true, err
	}
	usage := s.usage.snapshot()
	if usage.Iterations >= s.limits.MaxIterations {
		_, err := s.budgetReport(ctx, "execution budget exhausted")
		return true, err
	}
	if usage.ModelCalls >= s.limits.MaxModelCalls {
		_, err := s.budgetReport(ctx, "model call budget exhausted")
		return true, err
	}
	return false, nil
}
