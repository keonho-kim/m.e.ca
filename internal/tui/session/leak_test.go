// Purpose: Check all terminal workflows release their goroutines after package cleanup.
// Pattern: Package lifecycle test boundary.
// Usage: go test ./internal/tui/session.
// Related: internal/tui/session/app.go, internal/tui/session/app_test.go.
package session

import (
	"go.uber.org/goleak"
	"testing"
)

func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }
