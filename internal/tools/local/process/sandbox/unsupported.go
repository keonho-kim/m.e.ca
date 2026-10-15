//go:build !linux && !darwin

// Purpose: Fail closed when the host has no implemented filesystem isolation adapter.
// Pattern: Platform boundary.
// Usage: Unsupported builds return a sandbox error before shell execution.
// Related: internal/tools/local/process/sandbox/environment.go.
package sandbox

import (
	"context"
	"fmt"
)

func Prepare(context.Context, string, string, string, string) (Launch, error) {
	return Launch{}, fmt.Errorf("project filesystem isolation is unsupported on this platform")
}
