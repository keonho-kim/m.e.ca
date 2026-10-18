// Purpose: Preserve the inherited development environment while removing V and credential-agent control values.
// Pattern: Process environment projection.
// Usage: scratch supplies this environment to every sandbox launch.
// Related: internal/tools/local/process/sandbox/environment.go.
package sandbox

import (
	"os"
	"strings"
)

func hostEnvironment(scratch string) []string {
	var result []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "LD_") || strings.HasPrefix(key, "DYLD_") || strings.HasPrefix(key, "V_INTERNAL_") || strings.HasPrefix(key, "V_CODE_") || strings.HasPrefix(key, "V_TEST_") || strings.HasPrefix(key, "V_SANDBOX_") {
			continue
		}
		switch key {
		case "SSH_AUTH_SOCK", "SSH_AGENT_PID", "GPG_AGENT_INFO", "DBUS_SESSION_BUS_ADDRESS", "DOCKER_HOST", "CONTAINER_HOST", "TMPDIR", "TMP", "TEMP", "LD_PRELOAD", "LD_AUDIT", "DYLD_INSERT_LIBRARIES":
			continue
		}
		result = append(result, entry)
	}
	return append(result, "TMPDIR="+scratch, "TMP="+scratch, "TEMP="+scratch)
}
