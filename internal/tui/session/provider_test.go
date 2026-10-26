// Purpose: Verify inline provider selection sets the matching default URL and clears stale credentials.
// Pattern: Terminal model-settings test.
// Usage: go test ./internal/tui/session -run TestProviderSettings.
// Related: internal/tui/session/model_form_keys.go, internal/provider/connection/catalog.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"testing"
)

func TestProviderSettingsUseDeclaredDefaults(t *testing.T) {
	for target, provider := range connection.Providers() {
		a := fixture(t)
		a.openModel()
		a.modelForm.Credential.Set("private-key")
		for a.modelForm.Provider != target {
			a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
		}
		if a.page != "model" || a.modelForm.Endpoint.String() != provider.Endpoint {
			t.Fatalf("provider=%s url=%s", provider.Label, a.modelForm.Endpoint.String())
		}
		if target > 0 && a.modelForm.Credential.String() != "" {
			t.Fatal("provider change retained a typed credential")
		}
	}
}
