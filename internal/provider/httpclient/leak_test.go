// Purpose: Check generation, retry and cancellation tests release HTTP workers.
// Pattern: Package lifecycle test boundary.
// Usage: go test ./internal/provider/httpclient.
// Related: internal/provider/httpclient/client.go, internal/provider/transport/idle.go.
package httpclient

import (
	"go.uber.org/goleak"
	"testing"
)

func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }
