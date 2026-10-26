// Purpose: Recheck approval, Scope and file versions before one-shot execution.
// Pattern: One-shot operation lifecycle.
// Usage: Prepared tool executions expose Approval and Execute to the runtime.
// Related: internal/tools/local/prepare.go, internal/tools/local/textedit/edit.go.
package local

import (
	"context"
	"fmt"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	toolcatalog "github.com/keonho-kim/meca/internal/tools/local/catalog"
	"sync/atomic"
)

type operation struct {
	directories    []directoryTarget
	directoryPaths []string
	executor       *Executor
	settings       policy.Settings
	spec           toolcatalog.Specification
	input          toolcatalog.Arguments
	source         target
	original       version
	replacement    []byte
	editSpans      []toolapi.LineChange
	approval       toolapi.Approval
	executed       atomic.Bool
	environment    []string
	shell          shellSafety
}

func (op *operation) Approval() toolapi.Approval { return op.approval }

func (op *operation) Execute(ctx context.Context, approved bool) (toolapi.Result, error) {
	if err := ctx.Err(); err != nil {
		return toolapi.Result{}, err
	}
	if err := op.settings.Check(); err != nil {
		return toolapi.Result{}, err
	}
	if op.approval.Required && !approved {
		return toolapi.Result{}, fmt.Errorf("operation denied: approval required")
	}
	if !op.executed.CompareAndSwap(false, true) {
		return toolapi.Result{}, fmt.Errorf("prepared operation has already executed")
	}
	if op.spec.Effect == policy.Command {
		current, err := op.executor.analyzeCommand(ctx, op.input.Command, op.environment)
		if err != nil {
			return toolapi.Result{}, err
		}
		if !op.shell.same(current) {
			return toolapi.Result{}, &toolapi.Failure{Code: toolapi.StaleTarget, Detail: "command targets changed after preparation", Next: "Inspect the current targets and retry with the intended paths."}
		}
	} else if err := op.checkMutationSafety(); err != nil {
		return toolapi.Result{}, err
	}
	if op.spec.Name == "create_directory" {
		return op.createDirectories(ctx)
	}
	if op.spec.Effect != policy.Command {
		current, err := op.executor.resolve(op.settings.Scope, op.input.Path)
		if err != nil {
			return toolapi.Result{}, err
		}
		if current.absolute != op.source.absolute {
			return toolapi.Result{}, &toolapi.Failure{Code: toolapi.StaleTarget, Detail: "target changed after preparation", Next: "Inspect the intended path and its current contents before submitting the change again."}
		}
		if op.spec.Effect == policy.Change {
			version, err := snapshot(current)
			if err != nil {
				return toolapi.Result{}, err
			}
			if !sameVersion(op.original, version) {
				return toolapi.Result{}, &toolapi.Failure{Code: toolapi.StaleTarget, Detail: "file changed after preparation", Next: "Read the latest file and submit a change against its current contents."}
			}
		}
	}
	return op.run(ctx)
}
