// Purpose: Own cancellable task workers and execute only an explicitly approved Plan revision.
// Pattern: Use Case orchestration.
// Usage: Started by local chat submission or Plan approval; joined by Run.
// Related: internal/agent/run/run.go, internal/agent/plan/plan.go, internal/tui/session/app.go.
package session

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/keonho-kim/v-code/internal/agent"
	"github.com/keonho-kim/v-code/internal/agent/compact"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/config/prompts"
)

func (a *app) startTask(text string, approved *planning.Plan) {
	if a.busy {
		return
	}
	a.busy = true
	a.status = "Working"
	ctx, cancel := context.WithTimeout(a.ctx, time.Duration(a.opts.Config.Loop.MaxSeconds)*time.Second)
	a.taskCancel = cancel
	a.decision = make(chan bool, 1)
	runner := run.Runner{Generator: a.client, Tools: a.opts.Tools, Journal: a.opts.Store, Session: a.opts.Session.ID, Model: a.selection.Model, Provider: a.selection.Connection.ID, Thinking: a.selection.Thinking, Settings: a.settings, Redact: a.opts.Vault.Redact, MaskStream: a.opts.Vault.Stream, Limits: a.opts.Config.Loop, Usage: &run.Usage{}}
	history := append([]agent.Message(nil), a.history...)
	planningMode := a.planMode
	previous := a.plan
	var references []string
	for path := range a.browser.Selected {
		if _, err := a.opts.Tools.Reference(a.settings, path); err == nil {
			references = append(references, path)
		}
	}
	sort.Strings(references)
	if len(references) > 0 {
		text += "\nSelected path references (read only as needed):\n" + strings.Join(references, "\n")
	}
	decisions := a.decision
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		defer cancel()
		emit := func(event run.Event) error { return a.post(ctx, notice{Event: &event}) }
		approve := func(ctx context.Context, request agent.ApprovalRequest) (bool, error) {
			if err := a.post(ctx, notice{Approval: &request}); err != nil {
				return false, err
			}
			select {
			case value := <-decisions:
				return value, nil
			case <-ctx.Done():
				return false, ctx.Err()
			}
		}
		var result run.Result
		var err error
		var updated *planning.Plan
		templates, loadErr := prompts.Load(a.opts.State)
		if loadErr != nil {
			a.post(a.ctx, notice{Done: true, Err: loadErr})
			return
		}
		runner.BasePrompt = a.opts.Vault.Redact(templates.System)
		runner.Memory = &compact.Controller{Repository: a.opts.Store, Session: runner.Session, Model: runner.Model, Prompt: templates.Compaction, Settings: a.opts.Config.Compaction, Redact: a.opts.Vault.Redact}
		runner.ProjectInstructions, err = a.opts.Tools.ProjectInstructions(ctx)
		if err != nil {
			a.post(a.ctx, notice{Done: true, Err: err})
			return
		}
		if approved == nil {
			result, err = runner.Run(ctx, history, text, planningMode, approve, emit)
			if err == nil && planningMode && result.Status == run.PlanReady {
				proposal, parseErr := planning.ParsePlanProposal(result.Text)
				err = parseErr
				if err == nil {
					var p planning.Plan
					if previous != nil {
						p, err = planning.RevisePlan(*previous, proposal.Objective, proposal.Steps, time.Now())
					} else {
						p, err = planning.NewPlan("plan_"+rand.Text(), runner.Session, proposal.Objective, proposal.Steps, time.Now())
					}
					if err == nil {
						err = a.opts.Store.SavePlan(ctx, p)
						updated = &p
					}
				}
			}
		} else {
			p := *approved
			updated = &p
			for index, step := range p.Steps {
				p, err = planning.AdvanceStep(p, index, "running", time.Now())
				if err != nil {
					break
				}
				if err = a.opts.Store.SavePlan(ctx, p); err != nil {
					break
				}
				runner.Step = &compact.StepRef{PlanID: p.ID, Revision: p.Revision, Index: index, Description: step.Description, Objective: p.Objective}
				result, err = runner.Run(ctx, history, fmt.Sprintf("Execute approved Plan revision %d, step %d only. Objective: %s\nStep: %s", p.Revision, index+1, p.Objective, step.Description), false, approve, emit)
				history = result.Messages
				if err == nil && result.Status != run.Verified {
					err = fmt.Errorf("approved Plan step requires successful verification evidence")
				}

				state := "succeeded"
				if err != nil {
					state = "failed"
					if ctx.Err() != nil {
						state = "interrupted"
					}
				}
				next, transitionErr := planning.AdvanceStep(p, index, state, time.Now())
				if transitionErr != nil {
					err = transitionErr
					break
				}
				p = next
				saveErr := a.opts.Store.SavePlan(context.WithoutCancel(ctx), p)
				if saveErr != nil {
					err = saveErr
				}
				if err != nil {
					break
				}
			}
			updated = &p
		}
		if err != nil {
			recoveryErr := a.opts.Store.Recover(context.WithoutCancel(ctx), runner.Session)
			if recoveryErr != nil {
				err = errors.Join(err, recoveryErr)
			}
			restored, loadErr := a.opts.Store.Messages(context.WithoutCancel(ctx), runner.Session, 1000)
			if loadErr == nil {
				result.Messages = restored
			} else {
				err = errors.Join(err, loadErr)
			}
		}
		a.post(a.ctx, notice{Done: true, Err: err, Result: result, Plan: updated})
	}()
}
func (a *app) approvePlan() {
	if a.plan == nil || a.busy {
		return
	}
	if a.client == nil {
		a.status = "Select a model using !model"
		a.page = "chat"
		return
	}
	approved, err := planning.ApprovePlan(*a.plan, a.plan.Revision, time.Now())
	if err != nil {
		a.fail(err)
		return
	}
	if err = a.opts.Store.SavePlan(a.ctx, approved); err != nil {
		a.fail(err)
		return
	}
	a.plan = &approved
	a.planMode = false
	a.page = "chat"
	a.startTask("", &approved)
}
