//go:build !linux && !darwin

// Purpose: Refuse allocation without a supported native filesystem capacity check.
// Pattern: Platform boundary.
// Usage: Unsupported targets cannot bypass storage admission.
// Related: internal/storage/capacity/space_unix.go.
package capacity

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func CheckFree(string, uint64) error {
	return faults.Errorf(diagcatalog.CapacityFilesystemCapacityChecksRequireLinuxOrMacOS)
}
