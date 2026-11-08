// Purpose: Add cooperative tools while preserving ordinary local policy and owned command control.
// Pattern: Mode-scoped executor composition.
// Usage: Team request composition creates this adapter for one private participant lifetime.
// Related: internal/tools/team/coordination.go, internal/tools/team/local.go.
package team

import (
	"context"
	"sync"

	"github.com/keonho-kim/meca/internal/agent/team"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/skills"
	localcatalog "github.com/keonho-kim/meca/internal/tools/local/catalog"
	"github.com/keonho-kim/meca/internal/tools/team/approval"
	"github.com/keonho-kim/meca/internal/tools/team/catalog"
	"github.com/keonho-kim/meca/internal/tools/team/workspace"
)

type Local interface {
	toolapi.Executor
	PrepareAdmitted(context.Context, policy.Settings, toolapi.Call, func(context.Context) (func(), error)) (toolapi.Execution, error)
	toolapi.CommandExecutor
	toolapi.CommandNotifier
	ActiveCommand() string
	ParallelBatch(context.Context, policy.Settings, []toolapi.Call, int) (toolapi.BatchPlan, error)
	ProjectInstructions(context.Context) (string, error)
	Skills() *skills.Reader
	Close() error
}
type Executor struct {
	Local
	Peer         *team.Peer
	Workspace    *workspace.Access
	Refusals     *approval.Ledger
	Project      string
	mu           sync.Mutex
	commandPaths map[string][]string
}

// CheckUserDecision lets the runtime reject stale waves without projecting public notes mid-call.
func (e *Executor) CheckUserDecision() error { return e.Peer.CheckUserDecision() }

func (e *Executor) SetSkillImageInput(enabled bool) {
	if local, ok := e.Local.(interface{ SetSkillImageInput(bool) }); ok {
		local.SetSkillImageInput(enabled)
	}
}

func (e *Executor) Definitions() []toolapi.Definition {
	definitions := e.Local.Definitions()
	for i := range definitions {
		switch definitions[i].Name {
		case "write_file", "edit_file", "create_directory":
			definitions[i].Description += " Declare the target files or subdirectories with set_work before changing them. Respect the named teammate's ownership and inspect newer relevant changes before retrying a conflict."
		}
		if definitions[i].Name == "execute_command" {
			definitions[i].Description = localcatalog.CommandDescription("Independent commands run concurrently across teammates. Submit independent commands together; sequence dependent steps. Tell teammates your contribution before the first local tool. Use set_work before changing files; resolved mutation targets must belong to your current contribution. Only overlapping known mutation targets wait for shared access. Inspection commands do not require owning files. Coordinate commands that change the same dependency environment or generated outputs; arbitrary program effects cannot be inferred. Command controls manage only your own commands.")
		}
	}
	return append(definitions, catalog.Definitions()...)
}
func (e *Executor) Prepare(ctx context.Context, settings policy.Settings, call toolapi.Call) (toolapi.Execution, error) {
	if catalog.IsTool(call.Name) {
		input, err := catalog.Parse(call)
		if err != nil {
			return nil, err
		}
		return &coordinationOperation{executor: e, call: call, input: input}, nil
	}
	held, paths, err := e.reserve(ctx, call)
	if err != nil {
		return nil, err
	}
	prepared := &localOperation{executor: e, call: call, held: held, paths: paths}
	var op toolapi.Execution
	if call.Name == "execute_command" {
		op, err = e.Local.PrepareAdmitted(ctx, settings, call, prepared.admitCommand)
	} else {
		op, err = e.Local.Prepare(ctx, settings, call)
	}
	if err != nil {
		held.Release()
		return nil, err
	}
	if err := held.Check(op.Approval()); err != nil {
		held.Release()
		toolapi.Release(op)
		return nil, err
	}
	if proposal := op.Approval(); call.Name == "execute_command" && proposal.Command != nil {
		paths = proposal.Command.MutationTargets
	}
	if err == nil {
		err = e.Peer.CheckWork(paths)
	}
	if err != nil {
		held.Release()
		toolapi.Release(op)
		return nil, err
	}
	prepared.operation, prepared.paths = op, paths
	return prepared, nil
}
