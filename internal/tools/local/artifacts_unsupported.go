//go:build !unix

// Purpose: Report that captured-output leases require the supported Unix platforms.
// Pattern: Platform boundary.
// Usage: Executor initialization returns an explicit unsupported-platform error.
// Related: internal/tools/local/artifacts_unix.go, internal/tools/local/executor.go.
package local

import (
	"os"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func newArtifacts() (string, *os.File, error) {
	return "", nil, faults.Errorf(diagcatalog.LocalCapturedOutputLeasesRequireAUnixPlatform)
}
