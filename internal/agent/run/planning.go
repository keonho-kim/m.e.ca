// Purpose: Generate and persist an unapproved Plan draft or revision.
// Pattern: Planning use case.
// Usage: Session workers call Runner.DraftPlan for Plan mode.
// Related: internal/agent/plan/proposal.go, internal/agent/run/run.go, internal/storage/sqlite/plans.go.
package run

import (
	"context"
	"crypto/rand"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"time"
)

type PlanStore interface {
	SavePlan(context.Context, planning.Plan) error
}

func (r Runner) DraftPlan(ctx context.Context, text string, previous *planning.Plan, store PlanStore, approve Approver, emit func(Event) error) (*planning.Plan, error) {
	result, err := r.Run(ctx, text, true, approve, emit)
	if err != nil || result.Status != PlanReady {
		return nil, err
	}
	proposal, err := planning.ParsePlanProposal(result.Text)
	if err != nil {
		return nil, err
	}
	var draft planning.Plan
	if previous != nil {
		draft, err = planning.RevisePlan(*previous, proposal.Objective, proposal.Steps, time.Now())
	} else {
		draft, err = planning.NewPlan("plan_"+rand.Text(), r.Session, proposal.Objective, proposal.Steps, time.Now())
	}
	if err != nil {
		return nil, err
	}
	err = store.SavePlan(ctx, draft)
	return &draft, err
}
