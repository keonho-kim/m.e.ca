// Purpose: Verify LiteLLM defaults and provider-specific persisted connection identities.
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
	generic, err := New(OpenAICompatible, custom.Endpoint)
	if err != nil || generic.ID == custom.ID {
		t.Fatal("distinct Provider identities collapsed")
	}
}
