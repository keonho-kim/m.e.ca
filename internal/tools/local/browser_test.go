// Purpose: Verify browser tools obey ordinary scope, approval, credential and lazy-install boundaries.
// Pattern: Native preparation contract tests without browser downloads.
// Usage: go test ./internal/tools/local -run TestBrowser.
// Related: internal/tools/local/browser.go, internal/tools/local/catalog/browser.go.
package local

import (
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

func TestBrowserToolsOmittedWithoutInstallation(t *testing.T) {
	e, _ := testExecutor(t)
	var commands bool
	for _, definition := range e.Definitions() {
		switch definition.Name {
		case toolapi.BrowserAction, toolapi.BrowserInspect:
			t.Errorf("browser is uninstalled, yet %s was advertised", definition.Name)
		case "execute_command":
			commands = true
		}
	}
	if !commands {
		t.Fatal("browser availability changed the shell catalog")
	}
	if _, err := os.Lstat(filepath.Join(e.home, ".meca/browser")); !os.IsNotExist(err) {
		t.Fatalf("loading definitions touched browser state: %v", err)
	}
}

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
	remote, err := e.Prepare(t.Context(), settings, call("browser_action", `{"action":"open","url":"https://example.com/docs?q=browser#intro"}`))
	if err != nil {
		t.Fatal(err)
	}
	if approval := remote.Approval(); approval.Required || approval.UserRequired || len(approval.Reasons) != 0 {
		t.Fatal("opening an external URL should not prompt", approval)
	}
	if _, err := os.Stat(filepath.Join(e.home, ".meca/browser")); !os.IsNotExist(err) {
		t.Fatal("preparation eagerly created browser installation", err)
	}
	_, err = e.Prepare(t.Context(), policy.Settings{Scope: policy.System, Permission: policy.PermissionAllMighty}, call("browser_action", `{"action":"fill","target":"#entry","value":"plain"}`))
	if err == nil || !strings.Contains(err.Error(), diag.Text(diag.BrowserClosed)) {
		t.Fatal("interaction before open should fail before prompting", err)
	}
	for _, args := range []string{
		`{"action":"open","url":"https://example.com/?key=provider-secret"}`,
		`{"action":"open","url":"https://user:password@example.com"}`,
		`{"action":"fill","target":"#entry","value":"provider-secret"}`,
	} {
		if _, err := e.Prepare(t.Context(), settings, call("browser_action", args)); err == nil {
			t.Fatal("credential passed to browser")
		}
	}
	screenshot, err := e.Prepare(t.Context(), settings, call("browser_inspect", `{"view":"screenshot"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := screenshot.Execute(t.Context(), true); err == nil {
		t.Fatal("vision requirement ignored")
	}
}

func TestBrowserApprovalByOperation(t *testing.T) {
	for _, tc := range []struct {
		action   string
		interact bool
	}{
		{"click", true}, {"fill", true}, {"press", true}, {"select", true}, {"check", true}, {"uncheck", true}, {"hover", true}, {"back", true}, {"forward", true}, {"reload", true},
		{"open", false}, {"wait", false}, {"scroll", false}, {"viewport", false}, {"close", false}, {"", false},
	} {
		for _, url := range []string{"http://127.0.0.1", "https://example.com"} {
			for _, mode := range []policy.Permission{policy.PermissionDefault, policy.PermissionAllMighty} {
				op := browserOperation{address: url}
				op.settings.Permission = mode
				op.request.Action = tc.action
				if tc.action == "open" {
					op.request.URL = url
				}
				if tc.action == "" {
					op.request.View = "snapshot"
				}
				if err := op.browserApproval(); err != nil {
					t.Fatal(err)
				}
				direct := url == "https://example.com" && tc.interact
				prompt := direct || tc.interact && mode == policy.PermissionDefault
				if (op.approval.Required || op.approval.UserRequired) != prompt || op.approval.UserRequired != direct {
					t.Errorf("action=%q url=%s mode=%v prompt=%v direct=%v approval=%+v", tc.action, url, mode, prompt, direct, op.approval)
				}
			}
		}
	}
}
