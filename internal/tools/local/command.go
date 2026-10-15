// Purpose: Translate an approved shell operation into bounded process output and a sanitized tool result.
// Pattern: Tool Adapter.
// Usage: Called by the one-shot operation after Scope and Permission checks.
// Related: internal/tools/local/executor.go, internal/tools/local/process/run_unix.go, internal/tools/local/process_test.go.
package local

import (
	"context"
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	"github.com/keonho-kim/v-code/internal/tools/local/process"
	resultfmt "github.com/keonho-kim/v-code/internal/tools/result"
	"strings"
	"unicode/utf8"
)

func (op *operation) command(ctx context.Context) (toolapi.Result, error) {
	output, runErr := process.Run(ctx, op.executor.project, op.executor.home, op.executor.private, op.input.Command, op.settings.Scope == policy.Directory)
	output.Stdout = op.executor.redact(output.Stdout)
	output.Stderr = op.executor.redact(output.Stderr)
	result, err := resultfmt.Encode(output, fmt.Sprintf("Process %s; exit code %d", output.Status, output.ExitCode))
	if err != nil {
		return result, err
	}
	result.Process = &toolapi.ProcessOutcome{Status: output.Status, ExitCode: output.ExitCode, Truncated: output.Truncated}
	result.Failed = runErr != nil || output.ExitCode != 0 || output.Status != "exited"
	if result.Failed {
		result.Process.Diagnostic = commandDiagnostic(output.Stdout, output.Stderr)
	}
	if output.Status == "scope_unavailable" {
		result.Failure = &toolapi.Failure{Code: toolapi.ScopeUnavailable, Detail: output.Status, Next: "The command did not run. Resolve the host sandbox or runtime setup; do not widen Scope or retry outside containment.", Cause: runErr}
		if runErr != nil {
			result.Failure.Detail = op.executor.redact(runErr.Error())
		}
	} else if runErr != nil {
		result.Failure = toolapi.FailureOf(runErr)
	} else if result.Failed {
		result.Failure = &toolapi.Failure{Code: toolapi.ProcessFailed, Detail: result.Summary, Next: "Inspect the exact check output, fix its cause and rerun this check."}
	}
	return result, runErr
}

func commandDiagnostic(stdout, stderr string) string {
	text := strings.TrimSpace(stderr)
	if text == "" {
		text = strings.TrimSpace(stdout)
	}
	if len(text) <= 16*1024*1024 {
		return text
	}
	start := len(text) - 16*1024*1024
	for !utf8.RuneStart(text[start]) {
		start++
	}
	return "[diagnostic tail; see original captured output]\n" + text[start:]
}
