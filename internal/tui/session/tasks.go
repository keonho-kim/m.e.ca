// Purpose: Own task worker lifecycle and exchange runtime results and approval decisions.
// Pattern: Cancellable worker lifecycle.
// Usage: Started by local chat submission or chat input; joined by Run.
// Related: internal/agent/session/request.go, internal/agent/run/run.go, internal/tui/session/events.go.
package session

import (
	"context"
	"github.com/keonho-kim/v-code/internal/agent/run"
	tasksession "github.com/keonho-kim/v-code/internal/agent/session"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"sort"
	"strings"
)

func (a *app) startTask(text string) {
	if a.busy {
		return
	}
	a.output.Follow = true
	a.stream, a.thinking = "", ""
	a.tokensPerSecond = nil
	a.runSummary = nil
	a.stopNotice = ""
	a.busy = true
	a.startElapsed()
	a.agentActive = true
	a.stopRequested = false
	a.status = ""
	ctx, cancel := context.WithCancel(a.ctx)
	a.taskCancel = cancel
	a.decision = make(chan bool, 1)
	steering := make(chan string, 16)
	a.steering = steering
	inferenceSettings := a.opts.Config.InferenceSettings()
	runner := run.Runner{Inference: &inferenceSettings, Generator: a.client, Tools: a.opts.Tools, Journal: a.opts.Store, Session: a.opts.Session.ID, Model: a.selection.Model, Provider: a.selection.Connection.ID, Thinking: a.selection.Thinking, ThinkingSupported: a.selection.ThinkingSupported, ReasoningEffort: a.selection.Effort(), Settings: a.settings, Redact: a.opts.Vault.Redact, MaskStream: a.opts.Vault.Stream, Limits: a.opts.Config.Loop}
	runner.PollSteering = func() []string {
		var out []string
		for {
			select {
			case text := <-steering:
				out = append(out, text)
			default:
				return out
			}
		}
	}
	runner.ProjectRoot = a.opts.Session.ProjectRoot
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
	contextWindow := a.opts.Config.ContextLength
	if d := a.selection.Details; d != nil && d.ContextWindow > 0 {
		contextWindow = connection.EffectiveContextLength(contextWindow, d.ContextWindow)
	}
	store, loadPrompts, compaction := a.opts.Store, a.opts.LoadPrompts, a.opts.Config.Compaction.Settings
	instructions := a.opts.Tools.ProjectInstructions
	decisions := a.decision
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		defer cancel()
		emit := func(event run.Event) error {
			if event.Kind == "usage" || event.Statistics != nil {
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
		result, err := tasksession.Execute(ctx, tasksession.Request{Runner: runner, Store: store, LoadPrompts: loadPrompts, ContextWindow: contextWindow, Compaction: compaction, ProjectInstructions: instructions, Text: text}, approve, emit)
		a.post(a.ctx, notice{Done: true, Err: err, RunStatus: result.Status})
	}()
}
