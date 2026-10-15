// Purpose: Verify bounded structured reviews reject ambiguous or unsupported output.
// Pattern: Parser contract tests.
// Usage: go test ./internal/agent/conclave.
// Related: internal/agent/conclave/verdict.go.
package conclave

import "testing"

func TestStructuredVerdict(t *testing.T) {
	for _, text := range []string{`{"decision":"pass"}`, "```json\n{\"decision\":\"pass\"}\n```", `{"decision":"reject","findings":[{"path":"a","side":"after","line":1,"reason":"Wrong requested value"}]}`} {
		if _, err := Parse(text); err != nil {
			t.Fatalf("%s: %v", text, err)
		}
	}
	for _, text := range []string{"1", `{"decision":"reject"}`, `{"decision":"pass","decision":"reject"}`, `{"decision":"pass","unknown":true}`, `{"decision":"needs_context"}`, `{"decision":"pass"} {}`} {
		if _, err := Parse(text); err == nil {
			t.Fatalf("accepted %s", text)
		}
	}
}
