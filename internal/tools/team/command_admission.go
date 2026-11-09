// Purpose: Bind shared mutation reservations to the queued command's actual owned lifetime.
// Pattern: Process admission callback with effect-boundary checks.
// Usage: Executor.Prepare installs this callback on one concrete local shell operation.
// Related: internal/tools/team/executor.go, internal/tools/team/admission.go, internal/tools/local/command_operation.go.
package team

import (
	"context"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (op *localOperation) admitCommand(ctx context.Context) (func(), error) {
	e := op.executor
	held, err := e.reservePaths(ctx, op.call, op.paths, true)
	if err != nil {
		return nil, err
	}
	// The native operation rechecks its frozen AST and target identities after
	// this callback; repeating the same filesystem observation here adds no guard.
	if e.Refusals.Refused(op.operation.Approval(), &op.denialKey) {
		return held.Release, faults.Errorf(diagcatalog.TeamTheUserRefusedThisOperationDoNot)
	}
	if err := e.Peer.CheckUserDecision(); err != nil {
		return held.Release, err
	}
	return held.Release, e.Peer.CheckWork(op.paths)
}
