// Purpose: Verify ordinary inspection completes without decomposition or separate grounding calls.
// Pattern: Runtime behavior tests.
// Usage: go test ./internal/agent/run -run TestLean.
// Related: internal/agent/run/run.go, internal/agent/run/grounding.go.
package run

import (
	"context"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"testing"
)

func TestLeanReadCompletesWithoutExtraGeneration(t *testing.T) {
	g := &generator{replies: []string{nativeCall("read_file", map[string]any{"path": "a", "reason": "Inspect the requested file."}), "The file was inspected."}}
	e := &executor{}
	result, err := fixture(g, e, &journal{}).Run(context.Background(), "Read a", approve, emit)
	if err != nil || result.Status != Answered || e.executed != 1 || len(g.requests) != 2 || len(g.groundingRequests) != 0 {
		t.Fatalf("result=%+v executed=%d decisions=%d grounding=%d: %v", result, e.executed, len(g.requests), len(g.groundingRequests), err)
	}
}

func TestLeanReadFailureCanRecoverWithoutRequiredCheck(t *testing.T) {
	g := &generator{replies: []string{nativeCall("read_file", map[string]any{"path": "wrong"}), nativeCall("read_file", map[string]any{"path": "right"}), "The requested value is 40"}}
	e := &executor{outcomes: []toolapi.Result{{Failed: true, Content: []byte(`{"error":"wrong target"}`)}, {Content: []byte(`{"value":40}`)}}}
	j := &journal{}
	result, err := fixture(g, e, j).Run(context.Background(), "Read the value", approve, emit)
	if err != nil || result.Status != Answered || e.executed != 2 {
		t.Fatalf("%+v %v", result, err)
	}
}
