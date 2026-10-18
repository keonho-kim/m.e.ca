// Purpose: Verify inherited development settings are preserved and control credentials are excluded.
// Pattern: Environment projection contract tests.
// Usage: go test ./internal/tools/local/process/sandbox -run TestHostEnvironment.
// Related: internal/tools/local/process/sandbox/host_environment.go.
package sandbox

import (
	"slices"
	"testing"
)

func TestHostEnvironment(t *testing.T) {
	for key, value := range map[string]string{"HOME": "/real/home", "PATH": "/tools:/bin", "GOCACHE": "/cache/go", "UV_CACHE_DIR": "/cache/uv", "XDG_CONFIG_HOME": "/settings", "V_INTERNAL_SECRET": "hidden", "V_APPLICATION_SETTING": "preserved", "LD_LIBRARY_PATH": "/unsafe", "SSH_AUTH_SOCK": "/control"} {
		t.Setenv(key, value)
	}
	env := hostEnvironment("/scratch")
	for _, value := range []string{"HOME=/real/home", "PATH=/tools:/bin", "GOCACHE=/cache/go", "UV_CACHE_DIR=/cache/uv", "XDG_CONFIG_HOME=/settings", "TMPDIR=/scratch", "V_APPLICATION_SETTING=preserved"} {
		if !slices.Contains(env, value) {
			t.Fatal(value)
		}
	}
	for _, value := range []string{"V_INTERNAL_SECRET=hidden", "SSH_AUTH_SOCK=/control", "LD_LIBRARY_PATH=/unsafe"} {
		if slices.Contains(env, value) {
			t.Fatal(value)
		}
	}
}
