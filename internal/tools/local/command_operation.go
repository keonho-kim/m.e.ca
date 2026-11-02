// Purpose: Own a shell safety snapshot and recheck the exact command before process start.
// Pattern: Concrete prepared command execution.
// Usage: Execute rechecks shell policy before starting the prepared host command.
// Related: internal/tools/local/command.go, internal/tools/local/command_safety.go, internal/tools/local/prepare.go.
package local

import (
	"context"
	"fmt"
	"os"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

type commandOperation struct {
	operation
	environment []string
	shell       shellSafety
}

func (op *commandOperation) Execute(ctx context.Context, approved bool) (toolapi.Result, error) {
	if err := op.claim(ctx, approved); err != nil {
		return toolapi.Result{}, err
	}
	current, err := op.executor.analyzeCommand(ctx, op.input.Command, op.environment)
	if err != nil {
		return toolapi.Result{}, err
	}
	if !op.shell.same(current) {
		return toolapi.Result{}, &toolapi.Failure{Code: toolapi.StaleTarget, Detail: "command targets changed after preparation", Next: "Inspect the current targets and retry with the intended paths."}
	}
	return op.command(ctx)
}

func (op *commandOperation) prepareCommand(ctx context.Context) error {
	if op.settings.Scope != policy.System {
		return fmt.Errorf("host shell execution requires system Scope; directory confinement is unavailable")
	}
	op.approval.Target = op.input.Command
	op.environment = os.Environ()
	var err error
	op.shell, err = op.executor.analyzeCommand(ctx, op.input.Command, op.environment)
	if err != nil {
		return err
	}
	op.approval.Required = len(op.shell.decision.Reasons) > 0
	op.approval.UserRequired = op.shell.decision.UserRequired
	op.approval.Command = &toolapi.CommandApproval{Directory: op.executor.redact(op.executor.project)}
	for _, finding := range op.shell.findings {
		entry := toolapi.CommandFinding{Command: op.executor.redact(finding.Command)}
		for _, reason := range finding.Reasons {
			entry.Reasons = append(entry.Reasons, op.executor.redact(reason))
		}
		for _, path := range finding.Targets {
			entry.Targets = append(entry.Targets, op.executor.redact(path))
		}
		op.approval.Command.Findings = append(op.approval.Command.Findings, entry)
	}
	op.approval.Preview = op.executor.redact("Working directory: " + op.executor.project + "\n" + op.shell.decision.Description() + "\n\n" + op.input.Command)
	return nil
}
