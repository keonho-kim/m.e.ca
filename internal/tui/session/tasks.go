// Purpose: Own task worker lifecycle and exchange runtime results and approval decisions.
// Pattern: Cancellable worker lifecycle.
// Usage: Started by local chat submission or chat input; joined by Run.
// Related: internal/agent/session/request.go, internal/agent/run/run.go, internal/tui/session/events.go.
package session

import (
	"context"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	tasksession "github.com/keonho-kim/meca/internal/agent/session"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/commands/agentrequest"
)

func (a *app) startTask(text string) {
	if a.worker.active() {
		return
	}
	a.output.Follow = true
	a.stream, a.thinking = "", ""
	a.tokensPerSecond = nil
	a.runSummary = nil
	a.stopNotice = ""
	a.worker = workerAgent
	a.startElapsed()
	a.status = ""
	ctx, cancel := context.WithCancel(a.ctx)
	a.taskCancel = cancel
	a.decision = make(chan bool, 1)
	steering := make(chan string, 16)
	a.steering = steering
	request := agentrequest.New(agentrequest.Options{
		Config: a.opts.Config, Selection: a.selection, Session: a.opts.Session,
		Generator: a.client, Tools: a.opts.Tools, Store: a.opts.Store, Vault: a.opts.Vault,
		LoadPrompts: a.opts.LoadPrompts, Text: a.withSelectedPaths(text),
		PollSteering: func() []string {
			var out []string
			for {
				select {
				case text := <-steering:
					out = append(out, text)
				default:
					return out
				}
			}
		},
	})
	decisions := a.decision
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		defer cancel()
		delivery := runtimeDelivery{}
		emit := func(event runrecord.Event) error {
			if event.Kind == "stream" {
				return delivery.stream(ctx, a.events, event)
			}
			copy := event
			if event.Kind == "usage" || event.Statistics != nil {
				return delivery.post(a.ctx, a.events, notice{Event: &copy})
			}
			return delivery.post(ctx, a.events, notice{Event: &copy})
		}
		approve := func(ctx context.Context, request toolapi.Approval) (bool, error) {
			if err := delivery.post(ctx, a.events, notice{Approval: &request}); err != nil {
				return false, err
			}
			select {
			case value := <-decisions:
				return value, nil
			case <-ctx.Done():
				return false, ctx.Err()
			}
		}
		result, err := tasksession.Execute(ctx, request, approve, emit)
		delivery.post(a.ctx, a.events, notice{Done: true, Err: err, RunStatus: result.Status})
	}()
}
