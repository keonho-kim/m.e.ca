//go:build !linux && !darwin

// Purpose: Reject shell containment on platforms without filesystem identity guards.
// Pattern: Platform Adapter.
// Usage: Platform launch adapters require CheckTrees before execution.
// Related: internal/tools/local/filesystem/trees_unix.go.
package filesystem

import (
	"context"
	"fmt"
)

func (*Boundary) CheckTrees(context.Context) error {
	return fmt.Errorf("filesystem boundary unsupported on this platform")
}

func checkTarget(string, string) error {
	return fmt.Errorf("filesystem identity unsupported on this platform")
}
