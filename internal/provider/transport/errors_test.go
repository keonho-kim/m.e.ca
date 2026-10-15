// Purpose: Verify explicit error messages across native and compatible provider error envelopes.
// Pattern: Error contract tests.
// Usage: go test ./internal/provider/httpclient -run TestProviderError.
// Related: internal/provider/transport/errors.go.
package transport

import (
	"errors"
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

func TestContextCapacitySignalRequiresExplicitCode(t *testing.T) {
	for _, tc := range []struct {
		body string
		want bool
	}{
		{`{"error":{"code":"context_length_exceeded","message":"too long"}}`, true},
		{`{"error":{"code":"authentication_error","message":"context_length_exceeded"}}`, false},
		{`{"error":"context might be large"}`, false},
	} {
		var capacity interface{ ContextLimit() bool }
		err := ParseError(400, []byte(tc.body), "")
		got := errors.As(err, &capacity) && capacity.ContextLimit()
		if got != tc.want {
			t.Fatalf("%s: %v", tc.body, err)
		}
	}
}
