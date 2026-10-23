// Purpose: Apply shared project-boundary and protected-path rules to dedicated file changes.
// Pattern: Tool policy adapter.
// Usage: Prepared write, edit, move and directory operations call checkMutationSafety.
// Related: internal/policy/safety.go, internal/tools/local/candidate.go, internal/tools/local/directories.go.
package local

import "github.com/keonho-kim/v-code/internal/policy"

func (op *operation) checkMutationSafety() error {
	var decision policy.Safety
	check := func(path string, removal bool) error {
		result, err := policy.Mutation(op.executor.project, op.executor.home, path, removal)
		decision.Merge(result)
		return err
	}
	if op.spec.name == "create_directory" {
		for _, dir := range op.directories {
			if dir.info == nil {
				if err := check(dir.target.absolute, false); err != nil {
					return err
				}
			}
		}
	} else if op.spec.effect == policy.Change {
		if err := check(op.source.absolute, op.spec.name == "move_file"); err != nil {
			return err
		}
		if op.spec.name == "move_file" && op.destination.absolute != "" {
			if err := check(op.destination.absolute, false); err != nil {
				return err
			}
		}
	}
	if len(decision.Reasons) > 0 {
		op.approval.Required = true
	}
	return nil
}
