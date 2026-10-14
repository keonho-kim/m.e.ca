// Purpose: Verify all task phases retain the selected effort and record active generation settings.
// Pattern: Task behavior tests.
// Usage: go test ./internal/agent/run -run TestEffort.
// Related: internal/agent/run/context.go, internal/agent/run/journal.go.
package run

import (
	"context"
	"testing"
)

func TestEffortPropagatesThroughDecisionArgumentAndVerification(t *testing.T) {
	g := &generator{purposes: []string{"check"}, replies: []string{`{"tool":"read_file"}`, "AGENTS.md", "DONE\nChecked"}}
	j := &journal{}
	r := fixture(g, &executor{}, j)
	on := true
	r.Thinking = &on
	r.ReasoningEffort = "medium"
	if _, err := r.Run(context.Background(), "inspect", false, approve, emit); err != nil {
		t.Fatal(err)
	}
	if len(g.requests) != 3 {
		t.Fatalf("missing phases: %d", len(g.requests))
	}
	for _, request := range append(g.requests, g.purposeRequests...) {
		if request.ReasoningEffort != "medium" {
			t.Fatal("effort lost between phases")
		}
	}
	for _, message := range j.messages {
		if message.ReasoningEffort != "medium" {
			t.Fatal("active generation effort omitted from provenance")
		}
	}
}
