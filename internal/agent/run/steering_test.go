// Purpose: Verify direction changes join the real history after completed tool outcomes.
// Pattern: Deterministic input-queue regression tests.
// Usage: go test ./internal/agent/run -run TestSteering.
// Related: internal/agent/run/steering.go, internal/agent/run/run.go.
package run

import (
	"context"
	"testing"
)

func TestSteeringAfterToolOutcomeAndBeforeFinal(t *testing.T) {
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": "one"}), "Updated approach", "Finished"}}
	j := &journal{}
	r := fixture(g, &executor{}, j)
	delivered := false
	r.PollSteering = func() []string {
		if len(g.requests) == 1 && !delivered {
			delivered = true
			return []string{"Also include the endpoint"}
		}
		return nil
	}
	result, err := r.Run(context.Background(), "Implement a", approve, emit)
	if err != nil || result.Status != Answered {
		t.Fatalf("%+v %v", result, err)
	}
	if !delivered || len(g.requests) != 2 {
		t.Fatalf("requests=%d delivered=%v", len(g.requests), delivered)
	}
	var user, tool int
	for i, m := range j.messages {
		if m.Role == "tool" {
			tool = i
		}
		if m.Content == "Also include the endpoint" {
			user = i
		}
	}
	if user <= tool {
		t.Fatal("steering preceded the completed tool result")
	}
}

func TestSteeringArrivingDuringAnswerContinuesWithoutRegeneration(t *testing.T) {
	g := &generator{replies: []string{"First answer", "Updated answer"}}
	j := &journal{}
	r := fixture(g, &executor{}, j)
	delivered := false
	r.PollSteering = func() []string {
		if len(g.requests) == 1 && !delivered {
			delivered = true
			return []string{"Use the new requirement"}
		}
		return nil
	}
	result, err := r.Run(context.Background(), "Original requirement", approve, emit)
	if err != nil || result.Text != "Updated answer" || len(g.requests) != 2 {
		t.Fatalf("%+v %v", result, err)
	}
	if j.messages[1].Content != "First answer" || j.messages[2].Content != "Use the new requirement" {
		t.Fatalf("history reordered: %+v", j.messages)
	}
}
