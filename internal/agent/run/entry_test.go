// Purpose: Verify only a registered JSON entry can advance to literal argument collection.
// Pattern: Runtime trust-boundary test.
// Usage: go test ./internal/agent/run -run TestJSONEntry.
// Related: internal/agent/pfc/decision.go, internal/agent/run/action.go, internal/agent/run/context.go.
package run

import (
	"context"
	"strings"
	"testing"
)

func TestJSONEntryRejectsUnknownAndOneShotArguments(t *testing.T) {
	g := &generator{replies: []string{
		`{"tool":"missing_tool"}`,
		`{"tool":"write_file","arguments":{"path":"a","content":"bad"}}`,
		`{"tool":"read_file"}`, "a", "WAIT\nReview the file",
	}}
	e, j := &executor{}, &journal{}
	result, err := fixture(g, e, j).Run(context.Background(), "inspect", false, approve, emit)
	if err != nil || result.Status != NeedsInput || e.prepared != 1 || e.calls[0].Name != "read_file" {
		t.Fatalf("result=%+v calls=%+v err=%v", result, e.calls, err)
	}
	for _, index := range []int{0, 1, 2, 4} {
		question := g.requests[index].Messages[len(g.requests[index].Messages)-1].Content
		if !strings.Contains(question, "Available tools:") || !strings.Contains(question, "read_file:") {
			t.Fatal("decision lost the available tool catalog")
		}
	}
	if len(g.groundingRequests) != 1 {
		t.Fatal("invalid entries reached grounding")
	}
}
