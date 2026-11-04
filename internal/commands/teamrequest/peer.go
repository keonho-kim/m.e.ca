// Purpose: Open, configure and close exactly one private peer execution with its own bounded model scope.
// Pattern: Joined resource lifecycle at the application boundary.
// Usage: A request supplies execute to team.Run; runner callbacks keep its ordinary loop unchanged.
// Related: internal/commands/teamrequest/request.go, internal/agent/session/request.go.
package teamrequest

import (
	"context"
	"errors"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/run"
	agentsession "github.com/keonho-kim/meca/internal/agent/session"
	"github.com/keonho-kim/meca/internal/agent/team"
	"github.com/keonho-kim/meca/internal/agent/team/identity"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/commands/agentrequest"
	teamtools "github.com/keonho-kim/meca/internal/tools/team"
	"github.com/keonho-kim/meca/internal/tools/team/catalog"
)

func (r *request) execute(ctx context.Context, member identity.Member, input string, peer *team.Peer) (result run.Result, resultErr error) {
	child, err := r.service.Open(ctx, member.Session)
	if err != nil {
		return result, err
	}
	defer func() { resultErr = errors.Join(resultErr, child.Close()) }()
	local, err := r.service.NewTools()
	if err != nil {
		return result, err
	}
	r.service.track(member.ID, local)
	defer func() { r.service.track(member.ID, nil); resultErr = errors.Join(resultErr, local.Close()) }()
	tools := &teamtools.Executor{Local: local, Peer: peer, Workspace: r.access, Refusals: &r.refusals, Images: r.images, Project: r.options.Session.ProjectRoot}
	cursor, err := child.Store.ContextCursor(ctx, child.Session.ID)
	if err != nil {
		return result, err
	}
	if cursor == 0 {
		cursor = peer.InitialCursor()
	}
	peer.Cursor = cursor
	own := r.options
	own.Session, own.Store, own.Tools, own.Text = child.Session, child.Store, tools, input
	own.PollSteering = nil
	if scoped, ok := own.Generator.(interface {
		NewRunGeneratorWithBudget(int) (modelapi.Generator, func())
	}); ok {
		generator, release := scoped.NewRunGeneratorWithBudget(16 * 1024 * 1024 / team.MaxActive)
		own.Generator = generator
		defer release()
	}
	own.LoadPrompts = func() (prompt.Templates, error) {
		value := r.templates
		value.System += "\n\nYour peer name is " + member.ID + ". Speak as " + member.ID + ".\n"
		return value, nil
	}
	action := agentrequest.New(own)
	action.ProjectInstructions = func(ctx context.Context) (string, error) {
		r.instructions.Do(func() { r.projectInstructions, r.instructionsErr = tools.ProjectInstructions(ctx) })
		return r.projectInstructions, r.instructionsErr
	}
	action.Runner.PollContext, action.Runner.OnAnswer = peer.Inbox, peer.AfterAnswer
	action.Runner.BeforeDecision, action.Runner.ContextObservations = peer.BeginDecision, peer.ContextObservations
	action.Runner.Resume = true
	action.Runner.HideThinking = true
	action.Runner.AdmitModel, action.Runner.Deadline = peer.AdmitModel, peer.Deadline()
	emit := func(event runrecord.Event) error {
		if op := event.Operation; op != nil && op.Process != nil {
			copy := *op
			copy.AffectedPaths = tools.CommandTargets(op.Process.CommandID, op.Process.Status != "running")
			event.Operation = &copy
		}
		if event.Call != nil && catalog.IsTool(event.Call.Name) {
			event.Reference = nil
		}
		return peer.Emit(event)
	}
	approve := func(ctx context.Context, proposal toolapi.Approval) (bool, error) {
		return r.approve(ctx, member.ID, proposal)
	}
	return agentsession.Execute(ctx, action, approve, emit)
}
