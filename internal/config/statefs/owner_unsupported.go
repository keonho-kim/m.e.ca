//go:build !linux && !darwin

// Purpose: Reject private state initialization on unverified platforms.
// Pattern: Platform boundary.
// Usage: Used by configuration code outside Linux and macOS.
// Related: internal/config/statefs/owner_unix.go, cmd/meca/main.go.
package statefs

import (
	"os"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func checkOwner(os.FileInfo) error {
	return faults.Errorf(diagcatalog.StatefsPrivateStateIsSupportedOnlyOnLinux)
}
