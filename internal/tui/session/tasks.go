// Purpose: Own task worker lifecycle and exchange runtime results and approval decisions.
// Pattern: Cancellable worker lifecycle.
// Usage: Started by local chat submission or Plan approval; joined by Run.
// Related: internal/agent/run/planning.go, internal/agent/run/plan.go, internal/tui/session/events.go.
package session

import (
	"context"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/agent/run"
	tasksession "github.com/keonho-kim/v-code/internal/agent/session"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"sort"
	"strings"
)

func (a *app) startTask(text string, approved *planning.Plan) {
	if a.busy {
		return
	}
	a.stream, a.thinking = "", ""
	a.tokensPerSecond = nil
	a.stopNotice = ""
	a.busy = true
	a.startElapsed()
	a.agentActive = true
	a.stopRequested = false
	a.status = ""
	ctx, cancel := context.WithCancel(a.ctx)
	a.taskCancel = cancel
	a.decision = make(chan bool, 1)
	inferenceSettings := a.opts.Config.Inference
	runner := run.Runner{Inference: &inferenceSettings, Generator: a.client, Tools: a.opts.Tools, Journal: a.opts.Store, Session: a.opts.Session.ID, Model: a.selection.Model, Provider: a.selection.Connection.ID, Thinking: a.selection.Thinking, ThinkingSupported: a.selection.ThinkingSupported, ReasoningEffort: a.selection.Effort(), Settings: a.settings, Redact: a.opts.Vault.Redact, MaskStream: a.opts.Vault.Stream, Limits: a.opts.Config.Loop, Usage: &run.Usage{}}
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
	store, loadPrompts, compaction := a.opts.Store, a.opts.LoadPrompts, a.opts.Config.Compaction
	instructions := a.opts.Tools.ProjectInstructions
	decisions := a.decision
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		defer cancel()
		emit := func(event run.Event) error {
			if event.Kind == "usage" {
				return a.post(a.ctx, notice{Event: &event})
			}
			return a.post(ctx, notice{Event: &event})
		}
		approve := func(ctx context.Context, request toolapi.Approval) (bool, error) {
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
		updated, err := tasksession.Execute(ctx, tasksession.Request{Runner: runner, Store: store, LoadPrompts: loadPrompts, Compaction: compaction, ProjectInstructions: instructions, Text: text, Planning: planningMode, Previous: previous, Approved: approved}, approve, emit)
		a.post(a.ctx, notice{Done: true, Err: err, Plan: updated})
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
	approved, err := run.ApprovePlan(a.ctx, a.opts.Store, *a.plan)
	if err != nil {
		a.fail(err)
		return
	}
	a.plan = &approved
	a.planMode = false
	a.page = "chat"
	a.startTask("", &approved)
}
