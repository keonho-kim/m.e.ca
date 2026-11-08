// Purpose: Prevent agent sessions from running with unrestricted root privileges.
// Pattern: Pure startup policy.
// Usage: The composition root checks the effective user before session bootstrap.
// Related: cmd/meca/main.go, internal/commands/startup/user_test.go.
package startup

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func CheckAgentUser(effectiveUID int) error {
	if effectiveUID == 0 {
		return faults.Errorf(diagcatalog.StartupAgentSessionsCannotRunAsRootStart)
	}
	return nil
}
