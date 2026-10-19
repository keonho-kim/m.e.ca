// Purpose: Verify root privilege refusal without changing the test process identity.
// Pattern: Startup policy test.
// Usage: go test ./internal/commands/startup.
// Related: internal/commands/startup/user.go.
package startup

import "testing"

func TestAgentUser(t *testing.T) {
	if CheckAgentUser(0) == nil {
		t.Fatal("root allowed")
	}
	if err := CheckAgentUser(1000); err != nil {
		t.Fatal(err)
	}
}
