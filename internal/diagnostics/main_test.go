// Purpose: Dispatch the real output relay when tests launch their own executable.
// Pattern: Test process composition root.
// Usage: Diagnostics I/O tests run the same private worker as cmd/meca.
// Related: internal/diagnostics/stderr_relay.go, internal/diagnostics/stderr_backpressure_test.go.
package diagnostics

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if handled, code := RunStderrRelay(os.Args); handled {
		os.Exit(code)
	}
	os.Exit(m.Run())
}
