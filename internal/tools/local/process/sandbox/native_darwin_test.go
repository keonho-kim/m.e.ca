//go:build darwin

// Purpose: Keep native macOS isolation verification mandatory on supported hosts.
// Pattern: Platform test boundary.
// Usage: Filesystem isolation tests call requireSandbox before running platform assertions.
// Related: internal/tools/local/process/sandbox/filesystem_test.go.
package sandbox

import "testing"

func requireSandbox(t *testing.T) { t.Helper() }
