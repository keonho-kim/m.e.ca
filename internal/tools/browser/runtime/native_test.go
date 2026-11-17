// Purpose: Verify the pinned helper and current Stable Chrome can be installed and executed on the host.
// Pattern: Opt-in real download and executable smoke test.
// Usage: MECA_BROWSER_RUNTIME_STATE=/task/private/runtime go test ./internal/tools/browser/runtime -run TestNativeRuntime -v.
// Related: internal/tools/browser/runtime/install.go, internal/tools/browser/native_test.go.
package runtime

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
)

func TestNativeRuntime(t *testing.T) {
	state := os.Getenv("MECA_BROWSER_RUNTIME_STATE")
	if state == "" {
		t.Skip("explicit task directory required for runtime download")
	}
	helper, err := hostAgent()
	if err != nil {
		t.Fatal(err)
	}
	agent, err := install(t.Context(), downloadClient, state, helper)
	if err != nil {
		t.Fatal(err)
	}
	if output, err := exec.CommandContext(t.Context(), agent, "--version").CombinedOutput(); err != nil {
		t.Fatal(string(output), err)
	}
	chrome, err := latestStable(t.Context(), downloadClient, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatal(err)
	}
	browser, err := install(t.Context(), downloadClient, state, chrome)
	if err != nil {
		t.Fatal(err)
	}
	if output, err := exec.CommandContext(t.Context(), browser, "--version").CombinedOutput(); err != nil {
		t.Fatal(string(output), err)
	} else {
		t.Log(string(output))
	}
	t.Log("Browser:", browser)
}
