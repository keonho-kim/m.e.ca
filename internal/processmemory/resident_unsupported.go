//go:build !linux && !darwin

// Purpose: Report unavailable resident-memory measurement on unsupported operating systems.
// Pattern: Explicit platform boundary.
// Usage: Terminal status presents an unavailable value when native measurement is absent.
// Related: internal/processmemory/resident_linux.go, internal/processmemory/resident_darwin.go.
package processmemory

import "errors"

func ResidentBytes() (uint64, error) {
	return 0, errors.New("resident-memory measurement is unavailable on this operating system")
}
