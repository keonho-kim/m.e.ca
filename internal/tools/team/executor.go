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
	localcatalog "github.com/keonho-kim/meca/internal/tools/local/catalog"
	"github.com/keonho-kim/meca/internal/tools/team/approval"
	"github.com/keonho-kim/meca/internal/tools/team/catalog"
	"github.com/keonho-kim/meca/internal/tools/team/workspace"
)

type Local interface {
	toolapi.Executor
	toolapi.CommandExecutor
	toolapi.CommandNotifier
	CommandDone(string) (<-chan struct{}, error)
	ActiveCommand() string
	ProjectInstructions(context.Context) (string, error)
	Close() error
}
type Executor struct {
	Local        Local
	Peer         *team.Peer
	Workspace    *workspace.Access
	Refusals     *approval.Ledger
	Images       chan struct{}
	Project      string
	mu           sync.Mutex
	shell        *workspace.Lease
	commands     sync.WaitGroup
	commandPaths map[string][]string
}

// CheckUserDecision lets the runtime reject stale waves without projecting public notes mid-call.
func (e *Executor) CheckUserDecision() error { return e.Peer.CheckUserDecision() }

func (e *Executor) Definitions() []toolapi.Definition {
	definitions := e.Local.Definitions()
	for i := range definitions {
		switch definitions[i].Name {
		case "write_file", "edit_file", "create_directory":
			definitions[i].Description += " Declare the target files or subdirectories with set_work before changing them. Respect the named teammate's ownership and inspect newer relevant changes before retrying a conflict."
		}
		if definitions[i].Name == "execute_command" {
			definitions[i].Description = localcatalog.CommandDescription("Team shell commands execute one at a time across peers and are serialized with file changes. Tell teammates your contribution before the first local tool. Use set_work before changing files; resolved mutation targets must belong to your current contribution. Inspection commands do not require owning files. A call may wait for shared access; command controls remain available.")
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
	op, err := e.Local.Prepare(ctx, settings, call)
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
		paths, err = e.canonicalPaths(proposal.Command.MutationTargets, false)
	}
	if err == nil {
		err = e.Peer.CheckWork(paths)
	}
	if err != nil {
		held.Release()
		toolapi.Release(op)
		return nil, err
	}
	return &localOperation{executor: e, call: call, operation: op, held: held, paths: paths}, nil
}

func (e *Executor) ProjectInstructions(ctx context.Context) (string, error) {
	return e.Local.ProjectInstructions(ctx)
}
