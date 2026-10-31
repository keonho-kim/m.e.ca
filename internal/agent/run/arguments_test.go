// Purpose: Verify complete native arguments are corrected through paired results before any preparation.
// Pattern: Runtime trust-boundary tests.
// Usage: go test ./internal/agent/run -run TestNativeArgument.
// Related: internal/agent/run/action.go, internal/agent/run/tool_reply.go, internal/agent/tool/arguments.go.
package run

import (
	"context"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"strings"
	"testing"
)

func TestNativeArgumentCorrectionReturnsPairedFailure(t *testing.T) {
	g := &generator{replies: []string{nativeCall("inspect_fixture", map[string]any{"path": 12}), nativeCall("inspect_fixture", map[string]any{"path": "file.txt"}), "Inspected"}}
	e, j := &executor{}, &journal{}
	result, err := fixture(g, e, j).Run(context.Background(), "Read file.txt", approve, emit)
	if err != nil || result.Status != runrecord.Answered || e.prepared != 1 || len(g.requests) != 3 {
		t.Fatalf("result=%+v prepared=%d err=%v", result, e.prepared, err)
	}
	found := false
	for _, message := range g.requests[1].Messages {
		if message.Role == "tool" && message.ToolCallID == "call_1" && strings.Contains(message.Content, "argument validation failed") {
			found = true
		}
	}
	if !found {
		t.Fatal("native failure result missing from corrective request")
	}
}

func TestNativeArgumentFailureNeverPreparesATool(t *testing.T) {
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "file"}), "Need the file content"}}
	e, j := &executor{}, &journal{}
	result, err := fixture(g, e, j).Run(context.Background(), "Create file", approve, emit)
	if err != nil || e.prepared != 0 || len(g.requests) != 2 || result.Status != runrecord.Answered || !result.HadToolFailure {
		t.Fatalf("result=%+v prepared=%d err=%v", result, e.prepared, err)
	}
}

func TestNativeArgumentsHonorConfiguredBudget(t *testing.T) {
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": strings.Repeat("x", 2000)}), "Need a smaller change"}}
	e := &executor{}
	r := fixture(g, e, &journal{})
	r.Limits = DefaultLimits()
	r.Limits.MaxArgumentBytes = 1024
	result, err := r.Run(context.Background(), "create a", approve, emit)
	if err != nil || e.prepared != 0 || len(g.requests) != 2 || result.Status != runrecord.Answered || !result.HadToolFailure {
		t.Fatalf("prepared=%d result=%+v: %v", e.prepared, result, err)
	}
}
