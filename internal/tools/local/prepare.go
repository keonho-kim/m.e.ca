// Purpose: Prepare policy-controlled operations and bind approval previews to original file versions.
// Pattern: Approval preparation use case.
// Usage: The task runtime calls Executor.Prepare before requesting approval.
// Related: internal/tools/local/operation.go, internal/tools/local/paths.go, internal/tools/local/diff/diff.go.
package local

import (
	"context"
	"crypto/rand"
	toolcatalog "github.com/keonho-kim/meca/internal/tools/local/catalog"
	"os"

	"fmt"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"

	"github.com/keonho-kim/meca/internal/tools/local/textformat"
)

func (executor *Executor) Prepare(ctx context.Context, settings policy.Settings, call toolapi.Call) (toolapi.Execution, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	spec, input, err := toolcatalog.Parse(call)
	if err != nil {
		return nil, err
	}
	if err := settings.Check(); err != nil {
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
	op.approval = toolapi.Approval{ID: "operation_" + rand.Text(), Name: spec.Label}
	if spec.Effect == policy.Command {
		if settings.Scope != policy.System {
			return nil, fmt.Errorf("host shell execution requires system Scope; directory confinement is unavailable")
		}
		op.approval.Target = input.Command
		op.environment = os.Environ()
		op.shell, err = executor.analyzeCommand(ctx, input.Command, op.environment)
		if err != nil {
			return nil, err
		}
		op.approval.Required = len(op.shell.decision.Reasons) > 0
		op.approval.Command = &toolapi.CommandApproval{Directory: executor.redact(executor.project)}
		for _, finding := range op.shell.findings {
			entry := toolapi.CommandFinding{Command: executor.redact(finding.Command)}
			for _, reason := range finding.Reasons {
				entry.Reasons = append(entry.Reasons, executor.redact(reason))
			}
			for _, path := range finding.Targets {
				entry.Targets = append(entry.Targets, executor.redact(path))
			}
			op.approval.Command.Findings = append(op.approval.Command.Findings, entry)
		}
		op.approval.Preview = executor.redact("Working directory: " + executor.project + "\n" + op.shell.decision.Description() + "\n\n" + input.Command)
		return op, nil
	}
	if spec.Name == "create_directory" {
		if err := op.prepareDirectories(call); err != nil {
			return nil, err
		}
		return op, op.checkMutationSafety()
	}
	op.source, err = executor.resolve(settings.Scope, input.Path)
	if err != nil {
		return nil, err
	}
	op.approval.Target = op.source.absolute
	if err := op.checkMutationSafety(); err != nil {
		return nil, err
	}
	if err := op.prepareChange(call); err != nil {
		return nil, err
	}
	return op, op.checkMutationSafety()
}
