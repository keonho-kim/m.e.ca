//go:build linux || darwin

// Purpose: Preserve the original deadline cause when explicit cleanup cancels a child context.
// Pattern: Native cancellation regression test.
// Usage: Command lifetime tests run this under the race detector.
// Related: internal/tools/local/process/registry.go, internal/tools/local/process/run_unix.go.
package process

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestCommandPreservesDeadlineCancellationCause(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	var stdout, stderr cappedOutput
	pending, err := start(ctx, t.TempDir(), "sleep 60", os.Environ(), &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	cancel(context.DeadlineExceeded)
	output, err := pending.output(pending.wait())
	if !errors.Is(err, context.DeadlineExceeded) || output.Status != "timeout" {
		t.Fatalf("deadline cause replaced: %+v %v", output, err)
	}
}
