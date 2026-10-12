// Purpose: Translate an approved shell operation into bounded process output and a sanitized tool result.
// Pattern: Tool Adapter.
// Usage: Called by the one-shot operation after Scope and Permission checks.
// Related: internal/tools/local/executor.go, internal/tools/local/process/run_unix.go, internal/tools/local/process_test.go.
package local

import (
	"context"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent"
	"github.com/keonho-kim/v-code/internal/tools/local/process"
)

func (op *operation) command(ctx context.Context) (agent.ToolResult, error) {
	output, err := process.Run(ctx, op.executor.project, op.executor.home, op.input.Command, op.input.Timeout)
	if err != nil {
		return agent.ToolResult{}, err
	}
	output.Stdout = op.executor.redact(output.Stdout)
	output.Stderr = op.executor.redact(output.Stderr)
	result, err := encodeResult(output, fmt.Sprintf("Process exited with code %d", output.ExitCode))
	result.Failed = output.ExitCode != 0
	return result, err
}
