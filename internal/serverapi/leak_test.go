// Purpose: Verify streaming adapters and server request lifecycles release every background worker.
// Pattern: Package lifecycle test boundary.
// Usage: go test ./internal/serverapi.
// Related: internal/serverapi/transport.go, internal/serverapi/socket.go.
package serverapi

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }
