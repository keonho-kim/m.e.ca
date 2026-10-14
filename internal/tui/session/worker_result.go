// Purpose: Adopt completed worker results and restore the terminal interaction state.
// Pattern: Worker result state transition.
// Usage: The event loop invokes finishWorker when a task or connection completes.
// Related: internal/tui/session/events.go, internal/tui/session/connection.go.
package session

import (
	"context"
	"errors"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/tui/transcript"
)

func (a *app) finishWorker(n notice) {
	a.tokenCurrent = modelapi.TokenUsage{}
	if !a.planStopping && n.Err == nil && n.Effort != nil {
		n.Err = a.applyEffort(*n.Effort)
	}
	a.busy = false
	a.taskCancel = nil
	if n.Err != nil {
		a.output.Append(a.thinking, true)
		a.output.Append(transcript.Reply(a.stream), false)
	}
	a.stream, a.thinking = "", ""
	a.status = ""
	if a.planStopping {
		a.planStopping = false
		if n.Client != nil && n.Client != a.client {
			n.Client.Close()
		}
		if n.Plan != nil {
			a.plan = n.Plan
		}
		if err := a.leavePlan(); err != nil {
			a.fail(err)
		}
		if n.Err != nil && !errors.Is(n.Err, context.Canceled) {
			a.fail(n.Err)
		}
		return
	}
	if n.Plan != nil {
		a.plan = n.Plan
		a.planMode = n.Plan.Status == planning.AwaitingApproval || n.Plan.Status == planning.Planning
		a.output.Append(transcript.Reply(transcript.Plan(a.plan)), false)
		if a.planMode && a.plan.Status == planning.AwaitingApproval {
			a.page = "planapproval"
			a.choices = []string{"Approve", "Request changes"}
			a.choice = 0
		}
	}
	if n.Err != nil {
		a.status = a.opts.Vault.Redact(n.Err.Error())
	}
	if err := a.adoptConnection(n); err != nil {
		a.status = a.opts.Vault.Redact(err.Error())
	}
	if n.Err == nil && n.Models != nil {
		a.page = "models"
		a.choices = n.Models
		a.choice = 0
	}
	if a.proposal != nil {
		a.proposal = nil
		a.page = "chat"
	}
}
