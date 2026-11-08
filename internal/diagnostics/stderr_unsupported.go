//go:build !linux && !darwin

// Purpose: Reject diagnostic output on unsupported runtime platforms.
// Pattern: Explicit platform boundary.
// Usage: Supported process builds use stderr_unix.go instead.
// Related: internal/diagnostics/stderr_unix.go, cmd/meca/main.go.
package diagnostics

import (
	"io"

	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type Stderr struct{}

func OpenStderr() (*Stderr, error) {
	return nil, fault.Errorf(catalog.MecaMECACurrentlySupportsLinuxAndMacOS)
}
func (*Stderr) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func (*Stderr) Close() error              { return nil }
func emergencyWrite([]byte)               {}
