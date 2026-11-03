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
	"github.com/keonho-kim/meca/internal/tools/team/catalog"
	"github.com/keonho-kim/meca/internal/tools/team/workspace"
)

type Local interface {
	toolapi.Executor
	toolapi.CommandExecutor
	toolapi.CommandNotifier
	CommandDone(string) (<-chan struct{}, error)
	ProjectInstructions(context.Context) (string, error)
	Close() error
}
type Executor struct {
	Local     Local
	Peer      *team.Peer
	Workspace *workspace.Access
	Project   string
	mu        sync.Mutex
	shell     *workspace.Lease
	commands  sync.WaitGroup
}

func (e *Executor) Definitions() []toolapi.Definition {
	definitions := e.Local.Definitions()
	for i := range definitions {
		if definitions[i].Name == "execute_command" {
			definitions[i].Description = localcatalog.CommandDescription("Team shell commands execute one at a time across peers and are serialized with file changes. A call can wait for shared access before process start; command controls remain available.")
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
	held, err := e.reserve(ctx, call)
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
	return &localOperation{executor: e, call: call, operation: op, held: held}, nil
}

func (e *Executor) ProjectInstructions(ctx context.Context) (string, error) {
	return e.Local.ProjectInstructions(ctx)
}
