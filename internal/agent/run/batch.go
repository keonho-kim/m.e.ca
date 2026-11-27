// Purpose: Plan bounded host-tool waves and classify failures that stop code execution.
// Pattern: Serial owner with bounded parallel execution.
// Usage: Code dispatch uses these plans; one owner mutates runtime and journal state.
// Related: internal/agent/run/batch_wave.go, internal/agent/run/call_validation.go.
package run

import (
	"context"
	"errors"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/policy"

	// An executor opts in only when it can resolve aliases and reason about actual accesses.
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const maxParallelTools = 16

type parallelExecutor interface {
	ParallelBatch(context.Context, policy.Settings, []toolapi.Call, []toolapi.Call, int) (toolapi.BatchPlan, error)
}

func (s *execution) planWave(ctx context.Context, active, calls []validatedCall) (toolapi.BatchPlan, error) {
	idle := toolapi.BatchPlan{PreparationConcurrency: 1}
	if len(calls) == 0 || len(active) >= maxParallelTools {
		return idle, nil
	}
	planner, ok := s.runner.Tools.(parallelExecutor)
	if !ok {
		if len(active) != 0 {
			return idle, nil
		}
		return toolapi.BatchPlan{Count: 1, PreparationConcurrency: 1}, nil
	}
	var batch, running [maxParallelTools]toolapi.Call
	count := min(len(calls), maxParallelTools-len(active))
	for i := range count {
		batch[i] = calls[i].call
	}
	for i := range active {
		running[i] = active[i].call
	}
	plan, err := planner.ParallelBatch(ctx, s.runner.Settings, running[:len(active)], batch[:count], maxParallelTools)
	if err != nil {
		return plan, err
	}
	minimum := 0
	if len(active) == 0 {
		minimum = 1
	}
	if plan.Count < minimum || plan.Count > count || plan.PreparationConcurrency < 1 || plan.PreparationConcurrency > max(1, min(4, plan.Count)) {
		return plan, faults.Errorf(diagcatalog.RunInvalidParallelBatchBoundary)
	}
	return plan, nil
}

func fatalBatchError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, errApprovalDenied) || errors.Is(err, errRuntimeRecord) || faults.IsPanic(err) {
		return err
	}
	return nil
}
