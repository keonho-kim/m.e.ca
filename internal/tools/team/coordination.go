// Purpose: Execute team lifecycle, work and public-conversation operations through the owning peer.
// Pattern: One-shot collaboration operation.
// Usage: The run owner validates and commits these outcomes before session termination.
// Related: internal/tools/team/catalog/catalog.go, internal/agent/team/membership.go.
package team

import (
	"context"

	"sync/atomic"

	"github.com/keonho-kim/meca/internal/agent/team/source"
	"github.com/keonho-kim/meca/internal/tools/team/catalog"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	resultfmt "github.com/keonho-kim/meca/internal/tools/result"
)

type coordinationOperation struct {
	executor *Executor
	call     toolapi.Call
	input    catalog.Arguments
	used     atomic.Bool
}

func (op *coordinationOperation) Approval() toolapi.Approval {
	return toolapi.Approval{Name: op.call.Name, Target: op.input.Member}
}
func (op *coordinationOperation) Execute(ctx context.Context, _ bool) (toolapi.Result, error) {
	if err := ctx.Err(); err != nil {
		return toolapi.Result{}, err
	}
	if !op.used.CompareAndSwap(false, true) {
		return toolapi.Result{}, faults.Errorf(diagcatalog.TeamOperationAlreadyExecuted)
	}
	e := op.executor
	switch op.call.Name {
	case "add_team_member":
		value, err := e.Peer.AddMember(ctx, op.input.Member, op.input.Message)
		if err != nil {
			return toolapi.Result{}, err
		}
		return resultfmt.Encode(value, "Team member added or restored")
	case "list_team_members":
		page, err := e.Peer.Members(ctx, op.input.AfterMember)
		if err != nil {
			return toolapi.Result{}, err
		}
		return resultfmt.Encode(page, "Team members")
	case "read_team_messages":
		messages, err := e.Peer.ReadMessages(ctx, op.input.After)
		if err != nil {
			return toolapi.Result{}, err
		}
		return resultfmt.Encode(messages, "Public conversation")
	case "discuss":
		sequence, err := e.Peer.Discuss(ctx, op.input.Message, op.input.ReplyTo, op.input.Mentions)
		if err != nil {
			return toolapi.Result{}, err
		}
		result, err := resultfmt.Encode(map[string]int64{"sequence": sequence}, "Published to the team")
		result.Source = &source.Public{Sequence: sequence, Author: e.Peer.ID, Kind: "team_discussion"}
		return result, err
	case "set_work":
		paths, err := e.canonicalPaths(op.input.Paths, true)
		if err != nil {
			return toolapi.Result{}, err
		}
		if !e.Peer.HasWorkPaths(paths) {
			if e.Local.ActiveCommand() != "" {
				return toolapi.Result{}, faults.Errorf(diagcatalog.TeamSettleYourActiveCommandsBeforeChangingOr)
			}
		}
		work, err := e.Peer.SetWork(op.input.Summary, paths, op.input.Contract, op.input.ReplyTo)
		if err != nil {
			return toolapi.Result{}, err
		}
		result, err := resultfmt.Encode(struct {
			Member   string   `json:"member_name"`
			Paths    []string `json:"paths"`
			Sequence int64    `json:"sequence"`
			Revision uint64   `json:"work_revision"`
		}{work.Member, work.Paths, work.Sequence, work.Revision}, "Team contribution updated")
		result.Source = &source.Public{Sequence: work.Sequence, Author: e.Peer.ID, Kind: "team_work"}
		return result, err
	case "leave_team":
		if e.Local.ActiveCommand() != "" {
			return toolapi.Result{}, faults.Errorf(diagcatalog.TeamSettleActiveCommandsWithWaitCommandOr)
		}
		depart, err := e.Peer.Leave(ctx, op.input.Message)
		if err != nil {
			return toolapi.Result{}, err
		}
		status, next := "closed", "Your session is closing; its history remains available."
		if !depart {
			status = "report_required"
			next = "You are the last member. Read the latest public conversation and report directly to the user: completed changes, actual checks, remaining uncertainty. Respond normally without another leave_team. Use add_team_member only when the invitation criteria hold."
		}
		result, err := resultfmt.Encode(map[string]string{"status": status, "next": next}, "Contribution delivered")
		result.Depart = depart
		return result, err
	}
	return toolapi.Result{}, faults.Errorf(diagcatalog.CatalogUnknownCooperativeTool)
}
