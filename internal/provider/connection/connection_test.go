// Purpose: Verify provider defaults, API roots and distinct persisted connection identities.
// Pattern: Connection boundary tests.
// Usage: go test ./internal/provider/connection.
// Related: internal/provider/connection/connection.go, internal/storage/sqlite/selection.go.
package connection

import "testing"

func TestLiteLLMConnection(t *testing.T) {
	proxy, err := New(LiteLLM, "")
	if err != nil || proxy.Endpoint != "http://localhost:4000/v1" {
		t.Fatalf("default: %+v %v", proxy, err)
	}
	custom, err := New(LiteLLM, " https://example.test/gateway/v1/ ")
	if err != nil || custom.Endpoint != "https://example.test/gateway/v1" {
		t.Fatalf("custom: %+v %v", custom, err)
	}
	generic, err := New(OpenRouter, custom.Endpoint)
	if err != nil || generic.ID == custom.ID {
		t.Fatal("distinct Provider identities collapsed")
	}
}

func TestOpenRouterConnection(t *testing.T) {
	kind := ProviderKind("openrouter")
	for _, endpoint := range []string{"", "https://openrouter.ai", "https://openrouter.ai/api", "https://openrouter.ai/api/v1/"} {
		c, err := New(kind, endpoint)
		if err != nil || c.APIBase() != "https://openrouter.ai/api/v1" {
			t.Fatalf("endpoint %q: %+v %v", endpoint, c, err)
		}
	}
	found := false
	for _, provider := range Providers() {
		if provider.Kind == kind && provider.Label == "OpenRouter" {
			found = true
		}
		if provider.Kind == "openai-compatible" {
			t.Fatal("removed provider remains selectable")
		}
	}
	if !found {
		t.Fatal("OpenRouter is missing from provider selection")
	}
	if _, err := New("openai-compatible", ""); err == nil {
		t.Fatal("removed provider kind is still accepted")
	}
}
