// Purpose: Execute membership and public-history native operations through the owning peer.
// Pattern: One-shot collaboration operation.
// Usage: The run owner validates and commits these outcomes before session termination.
// Related: internal/tools/team/catalog/catalog.go, internal/agent/team/membership.go.
package team

import (
	"context"
	"fmt"
	"github.com/keonho-kim/meca/internal/tools/team/catalog"
	"sync/atomic"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	resultfmt "github.com/keonho-kim/meca/internal/tools/result"
)

type coordinationOperation struct {
	executor *Executor
	call     toolapi.Call
	input    catalog.Arguments
	used     atomic.Bool
}

func (op *coordinationOperation) Approval() toolapi.Approval {
	return toolapi.Approval{Name: op.call.Name, Target: op.input.Agent}
}
func (op *coordinationOperation) Execute(ctx context.Context, _ bool) (toolapi.Result, error) {
	if err := ctx.Err(); err != nil {
		return toolapi.Result{}, err
	}
	if !op.used.CompareAndSwap(false, true) {
		return toolapi.Result{}, fmt.Errorf("operation already executed")
	}
	e := op.executor
	switch op.call.Name {
	case "call_friend":
		value, err := e.Peer.Invite(ctx, op.input.Agent, op.input.Message)
		if err != nil {
			return toolapi.Result{}, err
		}
		return resultfmt.Encode(value, "Peer invitation processed")
	case "list_friends":
		page, err := e.Peer.Friends(ctx, op.input.AfterPeer)
		if err != nil {
			return toolapi.Result{}, err
		}
		return resultfmt.Encode(page, "Known peers")
	case "read_team_messages":
		messages, err := e.Peer.ReadMessages(ctx, op.input.After)
		if err != nil {
			return toolapi.Result{}, err
		}
		return resultfmt.Encode(messages, "Public conversation")
	case "say_goodbye":
		for _, command := range e.CommandActivity() {
			if command.Status == "running" {
				return toolapi.Result{}, fmt.Errorf("settle active commands with wait_command or cancel_command before leaving")
			}
		}
		depart, err := e.Peer.Goodbye(ctx, op.input.Message)
		if err != nil {
			return toolapi.Result{}, err
		}
		status, next := "closed", "Your session is closing; its history remains available."
		if !depart {
			status = "report_required"
			next = "You are the last peer. Read the latest public conversation and report directly to the user: completed changes, actual checks, remaining uncertainty. Respond normally without another goodbye. You may call a peer if more work is justified."
		}
		result, err := resultfmt.Encode(map[string]string{"status": status, "next": next}, "Contribution delivered")
		result.Depart = depart
		return result, err
	}
	return toolapi.Result{}, fmt.Errorf("unknown cooperative tool")
}
