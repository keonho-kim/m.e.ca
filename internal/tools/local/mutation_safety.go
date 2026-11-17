// Purpose: Apply shared project-boundary and protected-path rules to dedicated file changes.
// Pattern: Tool policy adapter.
// Usage: Concrete file and directory executions check each actual mutation path.
// Related: internal/policy/safety.go, internal/tools/local/candidate.go, internal/tools/local/directories.go.
package local

import (
	"github.com/keonho-kim/meca/internal/policy"
	"slices"
)

func (op *operation) checkMutationPath(path string) error {
	decision, err := policy.Mutation(op.executor.project, op.executor.home, path, false)
	if len(decision.Reasons) > 0 {
		op.approval.Required = true
		for _, reason := range decision.Reasons {
			masked := op.executor.redact(reason)
			if !slices.Contains(op.approval.Reasons, masked) {
				op.approval.Reasons = append(op.approval.Reasons, masked)
			}
		}
	}
	return err
}
