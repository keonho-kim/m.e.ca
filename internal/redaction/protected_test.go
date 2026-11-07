// Purpose: Verify lossless protected-state inspection and measure large native JSON scanning.
// Pattern: Secret-boundary tests and fixed-input benchmark.
// Usage: go test ./internal/config/credentials -run TestCheckProtected -bench BenchmarkCheckProtected.
// Related: internal/redaction/protected.go, internal/redaction/json.go.
package redaction

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCheckProtectedNestedStrings(t *testing.T) {
	vault := &Registry{}
	vault.Remember("provider-secret")
	for _, raw := range []string{
		`{"\u0070rovider-secret":"safe"}`,
		`{"arguments":"{\"path\":\"\\u0070rovider-secret\"}"}`,
		`{"opaque":"safe\u001bunsafe"}`,
	} {
		if err := CheckProtected([]byte(raw), vault.Redact); err == nil {
			t.Fatalf("protected JSON accepted: %s", raw)
		}
	}
	for _, raw := range []string{`{"number":9007199254740993,"text":"safe 日本語"}`, "opaque bytes", ""} {
		if err := CheckProtected([]byte(raw), vault.Redact); err != nil {
			t.Fatal(err)
		}
	}
}

func BenchmarkCheckProtected(b *testing.B) {
	vault := &Registry{}
	vault.Remember("provider-secret")
	arguments, _ := json.Marshal(map[string]string{"content": strings.Repeat("safe text ", 1<<17)})
	body, _ := json.Marshal(map[string]any{"type": "function_call", "arguments": string(arguments)})
	b.SetBytes(int64(len(body)))
	b.ReportAllocs()
	for b.Loop() {
		if err := CheckProtected(body, vault.Redact); err != nil {
			b.Fatal(err)
		}
	}
}
