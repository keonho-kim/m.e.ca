// Purpose: Prevent escaped JSON credentials from bypassing server output masking.
// Pattern: Secret redaction contract test.
// Usage: go test ./internal/config/credentials -run TestRedactJSON.
// Related: internal/redaction/json.go.
package redaction

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedactJSONEscapedSecretAndLargeInteger(t *testing.T) {
	vault := &Registry{}
	secret := "key-\"quoted\\secret"
	vault.Remember(secret)
	body, err := json.Marshal(map[string]any{"nested": []any{secret, map[string]string{secret: secret}}, "number": json.Number("9007199254740993")})
	if err != nil {
		t.Fatal(err)
	}
	result, err := RedactJSON(body, vault.Redact)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(result), "quoted") || !strings.Contains(string(result), "9007199254740993") || !json.Valid(result) {
		t.Fatal(string(result))
	}
	if _, err := RedactJSON([]byte(`{"unfinished":`), vault.Redact); err == nil {
		t.Fatal("partial JSON accepted")
	}
}
