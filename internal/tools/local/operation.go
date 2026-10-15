// Purpose: Recheck approval, Scope and file versions before one-shot execution.
// Pattern: One-shot operation lifecycle.
// Usage: Prepared tool executions expose Approval and Execute to the runtime.
// Related: internal/tools/local/prepare.go, internal/tools/local/file/read.go.
package local

import (
	"context"
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	"sync/atomic"
)

type operation struct {
	directories                  []directoryTarget
	directoryPaths               []string
	executor                     *Executor
	settings                     policy.Settings
	spec                         specification
	input                        arguments
	source, destination          target
	original, destinationVersion version
	replacement                  []byte
	editSpan                     *toolapi.LineChange
	approval                     toolapi.Approval
	executed                     atomic.Bool
}

func (op *operation) Approval() toolapi.Approval { return op.approval }

func (op *operation) Execute(ctx context.Context, approved bool) (toolapi.Result, error) {
	if err := ctx.Err(); err != nil {
		return toolapi.Result{}, err
	}
	required, err := policy.Decide(op.settings, op.spec.effect)
	if err != nil {
		return toolapi.Result{}, err
	}
	if required && !approved {
		return toolapi.Result{}, fmt.Errorf("operation denied: approval required")
	}
	if !op.executed.CompareAndSwap(false, true) {
		return toolapi.Result{}, fmt.Errorf("prepared operation has already executed")
	}
	if op.spec.name == "create_directory" {
		return op.createDirectories(ctx)
	}
	if op.spec.effect != policy.Command {
		current, err := op.executor.resolve(op.settings.Scope, op.input.Path)
		if err != nil {
			return toolapi.Result{}, err
		}
		if current.absolute != op.source.absolute {
			return toolapi.Result{}, &toolapi.Failure{Code: toolapi.StaleTarget, Detail: "target changed after preparation", Next: "Resolve and inspect the target again; request fresh approval."}
		}
		if op.spec.effect == policy.Change {
			version, err := snapshot(current)
			if err != nil {
				return toolapi.Result{}, err
			}
			if !sameVersion(op.original, version) {
				return toolapi.Result{}, &toolapi.Failure{Code: toolapi.StaleTarget, Detail: "file changed after approval", Next: "Read the latest file and prepare a new change with fresh approval."}
			}
		}
	}
	return op.run(ctx)
}
