// Purpose: Verify browser tools obey ordinary scope, approval, credential and lazy-install boundaries.
// Pattern: Native preparation contract tests without browser downloads.
// Usage: go test ./internal/tools/local -run TestBrowser.
// Related: internal/tools/local/browser.go, internal/tools/local/catalog/browser.go.
package local

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/policy"
)

func TestBrowserPreparation(t *testing.T) {
	e, _ := testExecutor(t)
	settings := policy.Settings{Scope: policy.System}
	open := call("browser_action", `{"action":"open","url":"http://127.0.0.1:3000"}`)
	if _, err := e.Prepare(t.Context(), policy.Settings{Scope: policy.Directory}, open); err == nil {
		t.Fatal("directory scope allowed browser")
	}
	if _, err := e.Prepare(t.Context(), settings, open); err != nil {
		t.Fatal(err)
	}
	remote, err := e.Prepare(t.Context(), policy.Settings{Scope: policy.System, Permission: policy.PermissionAllMighty}, call("browser_action", `{"action":"open","url":"https://example.com"}`))
	if err != nil || !remote.Approval().UserRequired {
		t.Fatal("external navigation bypassed approval", err)
	}
	if _, err := os.Stat(filepath.Join(e.home, ".meca/browser")); !os.IsNotExist(err) {
		t.Fatal("preparation eagerly created browser installation", err)
	}
	op, err := e.Prepare(t.Context(), policy.Settings{Scope: policy.System, Permission: policy.PermissionAllMighty}, call("browser_action", `{"action":"fill","target":"#entry","value":"plain"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !op.Approval().UserRequired {
		t.Fatal("external interaction bypassed direct approval")
	}
	if _, err := op.Execute(t.Context(), false); err == nil {
		t.Fatal("unapproved browser interaction executed")
	}
	if _, err := e.Prepare(t.Context(), settings, call("browser_action", `{"action":"fill","target":"#entry","value":"provider-secret"}`)); err == nil {
		t.Fatal("credential passed to browser")
	}
	screenshot, err := e.Prepare(t.Context(), settings, call("browser_inspect", `{"view":"screenshot"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := screenshot.Execute(t.Context(), true); err == nil {
		t.Fatal("vision requirement ignored")
	}
}
