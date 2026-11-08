// Purpose: Own a frozen file replacement and recheck its target and version before writing.
// Pattern: Concrete prepared file execution.
// Usage: Executor.Prepare returns this execution for write_file and edit_file only.
// Related: internal/tools/local/candidate.go, internal/tools/local/mutations.go, internal/tools/local/operation.go.
package local

import (
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"

	"context"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

type fileOperation struct {
	operation
	source      target
	original    version
	replacement []byte
	editSpans   []toolapi.LineChange
}

func (op *fileOperation) Execute(ctx context.Context, approved bool) (toolapi.Result, error) {
	if err := op.claim(ctx, approved); err != nil {
		return toolapi.Result{}, err
	}
	if err := op.mutable(); err != nil {
		return toolapi.Result{}, err
	}
	if err := op.checkMutationPath(op.source.absolute); err != nil {
		return toolapi.Result{}, err
	}
	current, err := op.currentTarget(op.source)
	if err != nil {
		return toolapi.Result{}, err
	}
	version, err := snapshot(current)
	if err != nil {
		return toolapi.Result{}, err
	}
	if !sameVersion(op.original, version) {
		return toolapi.Result{}, &faults.Failure{Code: diagcatalog.StaleTarget, Detail: diagcatalog.Text(diagcatalog.LocalFileChangedAfterPreparation), Next: diagcatalog.Text(diagcatalog.LocalReadTheLatestFileAndSubmitA)}
	}
	return op.mutate()
}
