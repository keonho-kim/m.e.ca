//go:build !linux && !darwin

// Purpose: Reject moves on platforms without the required pinned-parent rename contract.
// Pattern: Platform Adapter.
// Usage: File mutation dispatch returns an explicit unsupported error.
// Related: internal/tools/local/mutations.go.
package local

import "fmt"

func moveTargets(target, target) error { return fmt.Errorf("safe moves unsupported on this platform") }
