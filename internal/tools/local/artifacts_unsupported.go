//go:build !unix

// Purpose: Report that captured-output leases require the supported Unix platforms.
// Pattern: Platform boundary.
// Usage: Executor initialization returns an explicit unsupported-platform error.
// Related: internal/tools/local/artifacts_unix.go, internal/tools/local/executor.go.
package local

import (
	"fmt"
	"os"
)

func newArtifacts() (string, *os.File, error) {
	return "", nil, fmt.Errorf("captured-output leases require a Unix platform")
}
