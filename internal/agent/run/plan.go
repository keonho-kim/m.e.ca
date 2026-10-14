// Purpose: Execute approved Plan steps serially and persist their verified terminal states.
// Pattern: Approved Plan use case.
// Usage: Session workers call Runner.ExecutePlan with the approved revision.
// Related: internal/agent/plan/transitions.go, internal/agent/run/run.go, internal/agent/run/planning.go.
package run

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/goal"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
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
	goalState := goal.State{ID: "goal_" + rand.Text(), Request: p.Objective, Objective: p.Objective, Status: "pending"}
	for index, step := range p.Steps {
		goalState.Tasks = append(goalState.Tasks, goal.Task{ID: fmt.Sprintf("%s_task_%d", goalState.ID, index+1), Objective: step.Description, Kind: "work", Status: "pending"})
	}
	defer func() { err = r.finishGoalError(ctx, goalState, err) }()
	if err := r.recordGoal(ctx, goalState); err != nil {
		return p, err
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
		if err := goalState.Start(index); err != nil {
			return p, err
		}
		if err := r.recordGoal(ctx, goalState); err != nil {
			return p, err
		}
		r.assignment = &goal.Assignment{GoalID: goalState.ID, TaskID: goalState.Tasks[index].ID, GoalObjective: p.Objective, TaskObjective: step.Description, Kind: "work"}
		r.internalTask = true
		r.suppressStream = len(p.Steps) > 1
		r.Step = &planning.StepRef{PlanID: p.ID, Revision: p.Revision, Index: index, Description: step.Description, Objective: p.Objective}
		result, err := r.Run(ctx, fmt.Sprintf("Execute approved Plan revision %d, step %d only. Objective: %s\nStep: %s", p.Revision, index+1, p.Objective, step.Description), false, approve, func(event Event) error {
			if event.Kind == "response" {
				return nil
			}
			return emit(event)
		})
		if err == nil && result.Status != Verified {
			err = fmt.Errorf("approved Plan step requires successful verification evidence")
		}
		if finishErr := goalState.Finish(index, string(result.Status), result.Text, result.Evidence); finishErr != nil {
			return p, finishErr
		}
		if goalState.Status == "completed" {
			goalState.Status = "running"
		}
		flush, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		recordErr := r.recordGoal(flush, goalState)
		cancel()
		if recordErr != nil {
			return p, recordErr
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
	message := modelapi.Message{Role: "assistant", Content: r.Redact(text), ModelID: r.Model, ProviderID: r.Provider, Thinking: r.Thinking, ReasoningEffort: r.ReasoningEffort}
	if err := r.Journal.AppendMessage(ctx, r.Session, message); err != nil {
		return p, err
	}
	goalState.Status = "completed"
	if err := r.recordGoal(ctx, goalState); err != nil {
		return p, err
	}
	return p, emit(Event{Kind: "response", Text: text})
}
