// Purpose: Verify code continuations execute before unrelated native work finishes.
// Pattern: Real local tools with a channel-controlled operation dependency.
// Usage: go test -race ./internal/agent/run -run TestCodePipelines.
// Related: internal/agent/run/code_dispatch.go, internal/tools/local/parallel.go.
package run

import (
	"context"
	"sync"
	"testing"
	"time"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tools/local"
)

type pipelineTools struct {
	*local.Executor
	release chan struct{}
	once    sync.Once
}

type pipelineOperation struct {
	toolapi.Execution
	owner   *pipelineTools
	command string
}

func (e *pipelineTools) Prepare(ctx context.Context, settings policy.Settings, call toolapi.Call) (toolapi.Execution, error) {
	op, err := e.Executor.Prepare(ctx, settings, call)
	if err != nil {
		return nil, err
	}
	return &pipelineOperation{Execution: op, owner: e, command: callText(call, "command")}, nil
}

func (op *pipelineOperation) Execute(ctx context.Context, approved bool) (toolapi.Result, error) {
	if op.command == "printf slow" {
		select {
		case <-op.owner.release:
		case <-ctx.Done():
			return toolapi.Result{}, ctx.Err()
		}
	}
	if op.command == "printf continuation" {
		op.owner.once.Do(func() { close(op.owner.release) })
	}
	return op.Execution.Execute(ctx, approved)
}

func TestCodePipelinesFastResultBeforeEarlierSlowCall(t *testing.T) {
	r, native, _, journal, _ := realBatchRunner(t, nil)
	g := &generator{replies: []string{codeCall(`
		const slow=tools.execute_command({reason:"Observe slow operation",command:"printf slow"});
		const fast=await tools.execute_command({reason:"Observe fast operation",command:"printf fast"});
		if(fast.Stdout!=="fast") throw new Error("unexpected fast result");
		await tools.execute_command({reason:"Continue after fast result",command:"printf continuation"});
		return (await slow).Stdout;
	`), "Done"}}
	r.Generator = g
	r.Tools = &pipelineTools{Executor: native, release: make(chan struct{})}
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	result, err := r.Run(ctx, "pipeline independent work", approve, emit)
	if err != nil || result.Status != runrecord.Answered || result.HadToolFailure {
		t.Fatalf("continuation waited for slow sibling: %+v %v", result, err)
	}
	outcomes := innerOutcomes(journal)
	if len(outcomes) != 3 || callText(*outcomes[0].Call, "command") != "printf fast" {
		t.Fatalf("unexpected completed effects: %+v", outcomes)
	}
}
