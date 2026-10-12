// Purpose: Deliver worker results in order and apply them only in the session event loop.
// Pattern: Explicit event delivery and state transitions.
// Usage: Task and connection workers post notices; the terminal loop receives them.
// Related: internal/tui/session/app.go, internal/tui/session/tasks.go, internal/tui/session/model.go.
package session

import (
	"context"
	"github.com/keonho-kim/v-code/internal/agent"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/provider/httpclient"
	"github.com/keonho-kim/v-code/internal/storage/sqlite"
	"github.com/keonho-kim/v-code/internal/tools/local"
)

type notice struct {
	Event      *run.Event
	Approval   *agent.ApprovalRequest
	Done       bool
	Err        error
	Result     run.Result
	Plan       *planning.Plan
	Client     *httpclient.Client
	Models     []string
	Selection  *sqlite.Selection
	References []local.Reference
	Revision   int
}

func (a *app) post(ctx context.Context, n notice) error {
	select {
	case a.events <- n:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (a *app) receive(n notice) {
	if n.Revision > 0 {
		if n.Revision != a.refRevision {
			return
		}
		a.refs = n.References
		a.refIndex = 0
		if n.Err != nil {
			a.status = n.Err.Error()
		}
		return
	}
	if n.Approval != nil {
		a.proposal = n.Approval
		a.page = "approval"
		a.choices = []string{"Approve", "Reject"}
		a.choice = 0
		a.previewOffset = 0
		return
	}
	if n.Event != nil {
		e := n.Event
		if e.Kind == "tool_argument" {
			a.status = e.Text
			return
		}
		if e.Kind == "thinking" {
			a.output.Append(e.Thinking, true)
			return
		}
		if e.Kind == "stream" {
			a.stream, a.thinking = e.Text, e.Thinking
			if a.planMode {
				a.stream = ""
			}
		} else if e.Kind == "response" {
			a.stream, a.thinking = "", ""
			a.output.Append(e.Thinking, true)
			_, proposalErr := planning.ParsePlanProposal(e.Text)
			if !a.planMode || proposalErr != nil {
				a.output.Append(e.Text, false)
			}
		} else {
			a.output.Append(e.Text, false)
		}
		return
	}
	if n.Done {
		a.busy = false
		a.taskCancel = nil
		if n.Err != nil {
			a.output.Append(a.thinking, true)
			a.output.Append(a.stream, false)
		}
		a.stream, a.thinking = "", ""
		a.status = ""
		if n.Result.Messages != nil {
			a.history = n.Result.Messages
			if n.Result.Status == run.Answered {
				a.output.Append("Model answer; no tool execution.", false)
			}
			if n.Result.Status == run.Verified {
				a.output.Append("Verification evidence: "+n.Result.Evidence, false)
			}
		}
		if n.Plan != nil {
			a.plan = n.Plan
			a.planMode = n.Plan.Status == planning.AwaitingApproval || n.Plan.Status == planning.Planning
			a.output.Append(a.planText(), false)
			if a.planMode && a.plan.Status == planning.AwaitingApproval {
				a.page = "planapproval"
				a.choices = []string{"Approve", "Request changes"}
				a.choice = 0
			}
		}
		if n.Err != nil {
			a.status = a.opts.Vault.Redact(n.Err.Error())
			a.output.Append(a.status, false)
		}
		if n.Client != nil {
			if a.client != nil && a.client != n.Client {
				a.client.Close()
			}
			a.client = n.Client
		}
		if n.Selection != nil {
			if n.Selection.Model == "" {
				a.pendingConnection = n.Selection.Connection
			} else {
				a.selection = *n.Selection
				if a.selection.Model != "" {
					if err := a.opts.Store.SaveSelection(a.ctx, a.opts.Session.ID, a.selection); err != nil {
						a.status = err.Error()
					}
				}
			}
		}
		if n.Models != nil {
			a.page = "models"
			a.choices = n.Models
			a.choice = 0
		}
		if a.proposal != nil {
			a.proposal = nil
			a.page = "chat"
		}
	}
}
