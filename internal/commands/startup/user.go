// Purpose: Prevent agent sessions from running with unrestricted root privileges.
// Pattern: Pure startup policy.
// Usage: The composition root checks the effective user before session bootstrap.
// Related: cmd/v/main.go, internal/commands/startup/user_test.go.
package startup

import "fmt"

func CheckAgentUser(effectiveUID int) error {
	if effectiveUID == 0 {
		return fmt.Errorf("agent sessions cannot run as root; start V as a regular user")
	}
	return nil
}
