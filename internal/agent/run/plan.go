// Purpose: Execute approved Plan steps serially and persist their terminal states.
// Pattern: Approved Plan use case.
// Usage: Session workers call Runner.ExecutePlan with the approved revision.
// Related: internal/agent/plan/transitions.go, internal/agent/run/run.go, internal/agent/run/planning.go.
package run

import (
	"context"
	"fmt"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"strings"
	"time"
)

func (r Runner) ExecutePlan(ctx context.Context, p planning.Plan, store PlanStore, approve Approver, emit func(Event) error) (result planning.Plan, err error) {
	if store == nil || r.Journal == nil || r.Redact == nil || emit == nil {
		return p, fmt.Errorf("Plan execution dependencies are required")
	}
	if err := p.Check(); err != nil {
		return p, err
	}
	if p.Status != planning.Approved {
		return p, fmt.Errorf("Plan revision is not approved for execution")
	}
	if r.Usage == nil {
		r.Usage = &Usage{}
	}
	if r.observer == nil {
		observer, err := r.startMonitor(ctx)
		if err != nil {
			return p, err
		}
		r.observer = observer
		defer observer.Close()
	}
	var reports []string
	for index, step := range p.Steps {
		next, err := planning.AdvanceStep(p, index, "running", time.Now())
		if err != nil {
			return p, err
		}
		p = next
		if err = store.SavePlan(ctx, p); err != nil {
			return p, err
		}
		r.suppressStream = len(p.Steps) > 1
		r.Step = &planning.StepRef{PlanID: p.ID, Revision: p.Revision, Index: index, Description: step.Description, Objective: p.Objective}
		result, err := r.Run(ctx, fmt.Sprintf("Execute approved Plan revision %d, step %d only. Objective: %s\nStep: %s", p.Revision, index+1, p.Objective, step.Description), false, approve, func(event Event) error {
			if event.Kind == "response" {
				return nil
			}
			return emit(event)
		})
		if err == nil && result.Status != Answered {
			err = fmt.Errorf("approved Plan step stopped before its final report")
		}
		reports = append(reports, result.Text)
		state := "succeeded"
		if err != nil {
			state = "failed"
			if ctx.Err() != nil {
				state = "interrupted"
			}
		}
		next, transitionErr := planning.AdvanceStep(p, index, state, time.Now())
		if transitionErr != nil {
			return p, transitionErr
		}
		p = next
		if saveErr := store.SavePlan(context.WithoutCancel(ctx), p); saveErr != nil {
			return p, saveErr
		}
		if err != nil {
			return p, err
		}
	}
	text := strings.Join(reports, "\n\n")
	return p, emit(Event{Kind: "response", Text: text})
}
