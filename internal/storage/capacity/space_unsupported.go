//go:build !linux && !darwin

// Purpose: Refuse allocation without a supported native filesystem capacity check.
// Pattern: Platform boundary.
// Usage: Unsupported targets cannot bypass storage admission.
// Related: internal/storage/capacity/space_unix.go.
package capacity

import "fmt"

func CheckFree(string, uint64) error {
	return fmt.Errorf("filesystem capacity checks require Linux or macOS")
}
