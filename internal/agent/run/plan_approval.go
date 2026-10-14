// Purpose: Persist Plan mode entry and the exact revision explicitly approved by the user.
// Pattern: Plan use case.
// Usage: Terminal Plan commands invoke these use cases before executing work.
// Related: internal/agent/run/planning.go, internal/agent/plan/transitions.go.
package run

import (
	"context"
	"crypto/rand"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"time"
)

func BeginPlan(ctx context.Context, store PlanStore, session string) (planning.Plan, error) {
	p, err := planning.NewPlan("plan_"+rand.Text(), session, "Awaiting objective", []string{"Define the task objective"}, time.Now())
	if err != nil {
		return p, err
	}
	p.Status = planning.Planning
	return p, store.SavePlan(ctx, p)
}

func ApprovePlan(ctx context.Context, store PlanStore, p planning.Plan) (planning.Plan, error) {
	approved, err := planning.ApprovePlan(p, p.Revision, time.Now())
	if err != nil {
		return p, err
	}
	if err := store.SavePlan(ctx, approved); err != nil {
		return p, err
	}
	return approved, nil
}
