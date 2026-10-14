// Purpose: Verify explicit error messages across native and compatible provider error envelopes.
// Pattern: Error contract tests.
// Usage: go test ./internal/provider/httpclient -run TestProviderError.
// Related: internal/provider/transport/errors.go.
package transport

import (
	"strings"
	"testing"
)

func TestProviderErrorMessages(t *testing.T) {
	for _, body := range []string{
		`{"error":"context too large"}`,
		`{"error":{"message":"context too large","code":"context_length"}}`,
		`{"message":"context too large","code":400}`,
		`{"detail":"context too large"}`,
	} {
		if err := ParseError(400, []byte(body), ""); !strings.Contains(err.Error(), "context too large") {
			t.Fatalf("message lost: %v", err)
		}
	}
}
