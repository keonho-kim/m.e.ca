//go:build !linux && !darwin

// Purpose: Reject command execution on platforms without a verified process adapter.
// Pattern: Process Adapter.
// Usage: Selected by Go build constraints outside Linux and macOS.
// Related: internal/tools/local/process/run_unix.go, internal/tools/local/process/output.go.

package process

import (
	"context"
	"fmt"
)

func Run(context.Context, string, string, []string) (Output, error) {
	return Output{}, fmt.Errorf("command execution is unsupported on this platform")
}
