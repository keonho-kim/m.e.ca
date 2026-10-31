// Purpose: Exercise the native launch/wait boundary directly in process behavior tests.
// Pattern: Test fixture adapter.
// Usage: Native host, cancellation and output fixtures call Run.
// Related: internal/tools/local/process/run_unix.go, internal/tools/local/process/registry_test.go.
package process

import "context"

func Run(ctx context.Context, directory, command string, environment []string) (Output, error) {
	var stdout, stderr cappedOutput
	pending, err := start(ctx, directory, command, environment, &stdout, &stderr)
	if err != nil {
		return Output{Status: "execution_failed", ExitCode: -1}, err
	}
	return pending.output(pending.wait())
}
