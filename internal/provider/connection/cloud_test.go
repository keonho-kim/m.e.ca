// Purpose: Check first-party provider roots and full-strength credential binding identities.
// Pattern: Provider connection contract tests.
// Usage: go test ./internal/provider/connection -run TestCloud.
// Related: internal/provider/connection/catalog.go, internal/provider/connection/routes.go.
package connection

import "testing"

func TestCloudDefaultsAndCredentialScopes(t *testing.T) {
	for kind, endpoint := range map[ProviderKind]string{OpenAI: "https://api.openai.com/v1", Claude: "https://api.anthropic.com/v1", Gemini: "https://generativelanguage.googleapis.com/v1beta"} {
		c, err := New(kind, "")
		if err != nil || c.APIBase() != endpoint {
			t.Fatal(kind, c, err)
		}
		different, err := New(kind, "https://another.example/v1")
		if err != nil {
			t.Fatal(err)
		}
		if c.CredentialScope() == different.CredentialScope() || len(c.CredentialScope()) != 81 {
			t.Fatal("credential scope did not bind the full connection")
		}
	}
}
