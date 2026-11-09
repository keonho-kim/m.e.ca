//go:build !linux && !darwin

// Purpose: Reject application-state leases on unsupported platforms.
// Pattern: Platform boundary.
// Usage: Shared installation and startup code calls Lease.
// Related: internal/config/statefs/lease_unix.go.
package statefs

import (
	"os"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func Lease(*os.Root, bool) (*os.File, error) {
	return nil, faults.Errorf(diagcatalog.StatefsApplicationStateLeasesRequireLinuxOrMacOS)
}
