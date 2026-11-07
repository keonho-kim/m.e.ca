// Purpose: Supply a controlled native model and real shell adapter for team command overlap tests.
// Pattern: Test-only command barrier around the actual local executor.
// Usage: Team parallel tests hold command entry without sleeping or calling a model server.
// Related: internal/tools/team/parallel_test.go, internal/tools/local/executor.go, internal/agent/run/batch_wave.go.
package team

import (
	"context"
	"encoding/json"
	"fmt"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tools/local"
)

const commandsPerPeer = 16

type commandBatchModel struct{ calls int }

func (g *commandBatchModel) Generate(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
	g.calls++
	if g.calls > 1 {
		return emit(modelapi.Event{Delta: "Inspection complete", Done: true})
	}
	calls := make([]toolapi.Call, commandsPerPeer)
	for i := range calls {
		body, _ := json.Marshal(map[string]string{"command": fmt.Sprintf("printf 'command-%d\\n'", i)})
		calls[i] = toolapi.Call{ID: fmt.Sprint(i), Name: "execute_command", Arguments: body}
	}
	return emit(modelapi.Event{Calls: calls, Done: true})
}

type barrierCommands struct {
	*local.Executor
	entered chan<- string
	release <-chan struct{}
	member  string
}

func (l *barrierCommands) Prepare(ctx context.Context, settings policy.Settings, call toolapi.Call) (toolapi.Execution, error) {
	op, err := l.Executor.Prepare(ctx, settings, call)
	if err != nil {
		return nil, err
	}
	return barrierCommand{Execution: op, owner: l, id: call.ID}, nil
}

type barrierCommand struct {
	toolapi.Execution
	owner *barrierCommands
	id    string
}

func (op barrierCommand) Release() { toolapi.Release(op.Execution) }

func (op barrierCommand) Execute(ctx context.Context, approved bool) (toolapi.Result, error) {
	op.owner.entered <- op.owner.member + ":" + op.id
	select {
	case <-ctx.Done():
		return toolapi.Result{}, ctx.Err()
	case <-op.owner.release:
		return op.Execution.Execute(ctx, approved)
	}
}

func (l *barrierCommands) PrepareAdmitted(ctx context.Context, settings policy.Settings, call toolapi.Call, admit func(context.Context) (func(), error)) (toolapi.Execution, error) {
	op, err := l.Executor.PrepareAdmitted(ctx, settings, call, admit)
	if err != nil {
		return nil, err
	}
	return barrierCommand{Execution: op, owner: l, id: call.ID}, nil
}
