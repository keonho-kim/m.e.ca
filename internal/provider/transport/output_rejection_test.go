// Purpose: Verify request-correlated numeric evidence and prevent unrelated HTTP failures from triggering recovery.
// Pattern: Provider-envelope boundary cases.
// Usage: go test ./internal/provider/transport -run TestRejectedOutputEvidence.
// Related: internal/provider/transport/output_rejection.go.
package transport

import (
	"encoding/json"
	"testing"
)

func TestRejectedOutputEvidence(t *testing.T) {
	const exact = "This model's maximum context length is 100 tokens. However, you requested 75 output tokens and your prompt contains 50 input tokens, for a total of 125 tokens."
	const lower = "You requested 75 output tokens and your prompt contains at least 50 input tokens for a total of at least 125 tokens."
	for _, tc := range []struct {
		status, sent int
		code, msg    string
		want         bool
	}{
		{400, 75, "invalid_request_error", exact, true}, {422, 75, "context_length_exceeded", lower, true},
		{400, 74, "", lower, false}, {400, 75, "authentication_error", lower, false}, {401, 75, "", lower, false}, {503, 75, "", lower, false},
		{400, 75, "", "You requested 75 output tokens and your prompt contains 50 input tokens for a total of 126 tokens.", false},
		{400, 75, "", "Wrong tool arguments: " + lower, false},
	} {
		body, _ := json.Marshal(map[string]any{"error": map[string]string{"message": tc.msg, "code": tc.code}})
		evidence, ok := RejectedOutput(ParseError(tc.status, body, ""), tc.sent)
		if ok != tc.want {
			t.Errorf("status=%d code=%q sent=%d matched=%t", tc.status, tc.code, tc.sent, ok)
		}
		if ok && (evidence.Input != 50 || evidence.LowerBound != (tc.msg == lower) || tc.msg == exact && evidence.Capacity != 100) {
			t.Fatal(evidence)
		}
	}
}
