// Purpose: Verify opaque provider state contributes to context budgets without leaking protected content.
// Pattern: Context boundary tests.
// Usage: go test ./internal/agent/context -run TestProviderContinuation.
// Related: internal/agent/context/request.go, internal/agent/model/message.go.
package context

import (
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func TestProviderContinuationCountsTowardContextBudget(t *testing.T) {
	input := Input{Messages: []modelapi.Message{{Role: "assistant", Content: "done", ReasoningState: []byte(strings.Repeat("x", 100))}}, Redact: func(s string) string { return s }, MaxBytes: 50}
	if _, err := Build(input); err == nil {
		t.Fatal("continuation bypassed context byte budget")
	}
	input.MaxBytes = 200
	if _, err := Build(input); err != nil {
		t.Fatal(err)
	}
	input.Redact = func(s string) string { return strings.ReplaceAll(s, "x", "[redacted]") }
	if _, err := Build(input); err == nil {
		t.Fatal("protected continuation was sent or modified")
	}
}
