// Purpose: Verify the LiteLLM picker uses the same provider definition for its label and connection.
// Pattern: Terminal model-selection test.
// Usage: go test ./internal/tui/session -run TestLiteLLMProviderMenu.
// Related: internal/tui/session/model.go, internal/tui/session/menus.go, internal/provider/connection/connection.go.
package session

import (
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"testing"
)

func TestLiteLLMProviderMenu(t *testing.T) {
	a := fixture(t)
	a.openModel()
	a.choose()
	found := false
	for index, label := range a.choices {
		if label != "LiteLLM" {
			continue
		}
		found = true
		a.choice = index
		a.choose()
		if a.page != "endpoint" || a.menuInput.String() != "http://localhost:4000/v1" || modelconn.Providers()[a.connectionKind].Kind != modelconn.LiteLLM {
			t.Fatalf("incorrect LiteLLM selection: %s %q", a.page, a.menuInput.String())
		}
		break
	}
	if !found {
		t.Fatal("LiteLLM not available in provider picker")
	}
}
