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
	"github.com/keonho-kim/v-code/internal/tools/local/textformat"
	"strings"
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
	texts := []string{input.Content, input.Command}
	for _, edit := range input.Edits {
		texts = append(texts, edit.OldText, edit.NewText)
	}
	for _, text := range texts {
		if textformat.RedactSource(text, executor.redact) != text {
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
		op.approval.Preview = executor.redact("Working directory: " + executor.project + "\nWritable roots: " + strings.Join(executor.boundary.Writes(), ", ") + "\n\n" + input.Command)
		return op, nil
	}
	if spec.name == "create_directory" {
		return op, op.prepareDirectories(call)
	}
	op.source, err = executor.writeTarget(settings.Scope, input.Path)
	if err != nil {
		return nil, err
	}
	op.approval.Target = op.source.absolute
	if err := op.prepareChange(call); err != nil {
		return nil, err
	}
	return op, nil
}
