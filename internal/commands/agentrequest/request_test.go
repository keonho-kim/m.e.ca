// Purpose: Verify both application entry points receive a copied permission setting.
// Pattern: Request composition contract test.
// Usage: go test ./internal/commands/agentrequest.
// Related: internal/commands/agentrequest/request.go, internal/config/settings.go.
package agentrequest

import (
	"testing"

	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tools/local"
)

func TestPermissionSnapshot(t *testing.T) {
	configuration := config.Defaults()
	configuration.Permission = policy.PermissionAllMighty
	home := t.TempDir()
	tools, err := local.NewExecutor(t.TempDir(), home, home+"/.meca", func(text string) string { return text })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tools.Close() })
	request := New(Options{Config: configuration, Vault: &credentials.Vault{}, Tools: tools})
	configuration.Permission = policy.PermissionDefault
	if request.Runner.Settings.Permission != policy.PermissionAllMighty || request.Runner.Settings.Scope != policy.System {
		t.Fatal("request did not retain the selected execution policy")
	}
}
