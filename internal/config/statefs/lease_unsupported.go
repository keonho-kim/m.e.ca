//go:build !linux && !darwin

// Purpose: Reject application-state leases on unsupported platforms.
// Pattern: Platform boundary.
// Usage: Shared installation and startup code calls Lease.
// Related: internal/config/statefs/lease_unix.go.
package statefs

import (
	"fmt"
	"os"
)

func Lease(*os.Root, bool) (*os.File, error) {
	return nil, fmt.Errorf("application state leases require Linux or macOS")
}
