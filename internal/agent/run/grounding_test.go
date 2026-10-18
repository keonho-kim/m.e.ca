// Purpose: Verify bound intent is durable before execution and survives in native history.
// Pattern: Runtime trust-boundary tests.
// Usage: go test ./internal/agent/run -run TestGrounding.
// Related: internal/agent/run/grounding.go, internal/agent/tool/intent.go.
package run

import (
	"context"
	"strings"
	"testing"
)

func TestGroundingMissingReasonNeverPrepares(t *testing.T) {
	g := &generator{replies: []string{`native:{"id":"c","name":"inspect_fixture","arguments":{"path":"a"}}`, "Need a reason."}}
	e := &executor{}
	result, err := fixture(g, e, &journal{}).Run(context.Background(), "inspect", approve, emit)
	if err != nil || e.prepared != 0 || !result.HadToolFailure {
		t.Fatalf("%+v prepared=%d: %v", result, e.prepared, err)
	}
}
func TestGroundingPersistenceFailureBlocksPreparation(t *testing.T) {
	g := &generator{replies: []string{nativeCall("inspect_fixture", map[string]any{"path": "a"})}}
	e := &executor{}
	_, err := fixture(g, e, &journal{failKind: "self_grounding"}).Run(context.Background(), "inspect", approve, emit)
	if err == nil || e.prepared != 0 {
		t.Fatalf("prepared=%d err=%v", e.prepared, err)
	}
}
func TestGroundingPersistsAndStaysInContext(t *testing.T) {
	g := &generator{replies: []string{nativeCall("inspect_fixture", map[string]any{"path": "a"}), "Observed a."}}
	e, j := &executor{}, &journal{}
	_, err := fixture(g, e, j).Run(context.Background(), "inspect", approve, emit)
	if err != nil {
		t.Fatal(err)
	}
	grounded := false
	for _, event := range j.events {
		if event.Grounding != nil {
			grounded = event.Grounding.Matches(e.calls[0])
		}
		if event.Kind == "tool_started" && !grounded {
			t.Fatal("execution before grounding")
		}
	}
	if !grounded || len(g.requests) != 2 {
		t.Fatal("intent missing or extra model call")
	}
	found := false
	for _, m := range g.requests[1].Messages {
		for _, c := range m.Calls {
			found = found || strings.Contains(string(c.Arguments), `"reason"`)
		}
	}
	if !found {
		t.Fatal("native intent lost from history")
	}
}
