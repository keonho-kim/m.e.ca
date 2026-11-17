//go:build !darwin && !linux

// Purpose: Reject unobservable browser memory admission on unsupported systems.
// Pattern: Explicit platform boundary.
// Usage: Browser admission calls Pressure before allocating external resources.
// Related: internal/processmemory/pressure_darwin.go, internal/processmemory/pressure_linux.go.
package processmemory

import (
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func Pressure() (bool, error) {
	return false, fault.Errorf(diag.ProcessmemoryResidentMemoryMeasurementIsUnavailableOnThis)
}
