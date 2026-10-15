// Purpose: Prepare policy-controlled operations and bind approval previews to original file versions.
// Pattern: Approval preparation use case.
// Usage: The task runtime calls Executor.Prepare before requesting approval.
// Related: internal/tools/local/operation.go, internal/tools/local/paths.go, internal/tools/local/diff/diff.go.
package local

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	"github.com/keonho-kim/v-code/internal/policy/shell"
)

func (executor *Executor) Prepare(ctx context.Context, settings policy.Settings, call toolapi.Call) (toolapi.Execution, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	spec, input, err := parseArguments(call)
	if err != nil {
		return nil, err
	}
	required, err := policy.Decide(settings, spec.effect)
	if err != nil {
		return nil, err
	}
	for _, text := range []string{input.Content, input.NewText, input.Command} {
		if executor.redact(text) != text {
			return nil, fmt.Errorf("tool arguments contain a protected credential")
		}
	}
	op := &operation{executor: executor, settings: settings, spec: spec, input: input}
	op.approval = toolapi.Approval{ID: "operation_" + rand.Text(), Name: spec.label, Required: required}
	if spec.effect == policy.Command {
		if err := shell.Check(input.Command); err != nil {
			var dedicated *shell.Dedicated
			if errors.As(err, &dedicated) {
				return nil, &toolapi.Failure{Code: "use_dedicated_tool", Detail: err.Error(), Next: dedicatedAdvice(dedicated.Tool)}
			}
			return nil, &toolapi.Failure{Code: toolapi.InvalidArgument, Detail: err.Error(), Next: "Return a valid bounded POSIX shell command. No command was executed."}
		}
		op.approval.Target = input.Command
		op.approval.Preview = input.Command
		return op, nil
	}
	if spec.name == "create_directory" {
		return op, op.prepareDirectories(call)
	}
	op.source, err = executor.resolve(settings.Scope, input.Path)
	if err != nil {
		return nil, err
	}
	op.approval.Target = op.source.absolute
	if spec.effect == policy.Read {
		if _, err := op.source.root.Stat(op.source.relative); err != nil {
			return nil, err
		}
		return op, nil
	}
	if err := op.prepareChange(call); err != nil {
		return nil, err
	}
	return op, nil
}
