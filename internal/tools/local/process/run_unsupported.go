//go:build !linux && !darwin

// Purpose: Reject command execution on platforms without a verified process adapter.
// Pattern: Process Adapter.
// Usage: Selected by Go build constraints outside Linux and macOS.
// Related: internal/tools/local/process/run_unix.go, internal/tools/local/process/output.go.

package process

import (
	"context"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func start(context.Context, string, string, []string, *cappedOutput, *cappedOutput) (startedCommand, error) {
	return startedCommand{}, faults.Errorf(diagcatalog.ProcessCommandExecutionIsUnsupportedOnThisPlatform)
}
