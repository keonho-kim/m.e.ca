// Purpose: Check tool waves and model cancellation leave no execution goroutines behind.
// Pattern: Package lifecycle test boundary.
// Usage: go test ./internal/agent/run.
// Related: internal/agent/run/batch_wave.go, internal/agent/run/usage.go.
package run

import (
	"go.uber.org/goleak"
	"testing"
)

func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }
