// Purpose: Verify complete native arguments are corrected through paired results before any preparation.
// Pattern: Runtime trust-boundary tests.
// Usage: go test ./internal/agent/run -run 'TestNativeArgument|TestGreeting'.
// Related: internal/agent/run/action.go, internal/agent/run/tool_reply.go, internal/agent/tool/arguments.go.
package run

import (
	"context"
	"strings"
	"testing"
)

func TestNativeArgumentCorrectionReturnsPairedFailure(t *testing.T) {
	g := &generator{replies: []string{nativeCall("inspect_fixture", map[string]any{"path": 12}), nativeCall("inspect_fixture", map[string]any{"path": "file.txt"}), "Inspected"}}
	e, j := &executor{}, &journal{}
	result, err := fixture(g, e, j).Run(context.Background(), "Read file.txt", approve, emit)
	if err != nil || result.Status != Answered || e.prepared != 1 || len(g.requests) != 3 {
		t.Fatalf("result=%+v prepared=%d err=%v", result, e.prepared, err)
	}
	found := false
	for _, message := range g.requests[1].Messages {
		if message.Role == "tool" && message.ToolCallID == "call_1" && strings.Contains(message.Content, "Argument validation failed") {
			found = true
		}
	}
	if !found {
		t.Fatal("native failure result missing from corrective request")
	}
}

func TestNativeArgumentFailureNeverPreparesATool(t *testing.T) {
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "file"}), "Need the file content", "Need the file content"}}
	e, j := &executor{}, &journal{}
	result, err := fixture(g, e, j).Run(context.Background(), "Create file", approve, emit)
	if err != nil || e.prepared != 0 || result.Status != Answered || !result.HadToolFailure {
		t.Fatalf("result=%+v prepared=%d err=%v", result, e.prepared, err)
	}
}

func TestGreetingDoesNotRequireATool(t *testing.T) {
	g, e, j := &generator{replies: []string{"Hello"}}, &executor{}, &journal{}
	result, err := fixture(g, e, j).Run(context.Background(), "Hello", approve, emit)
	if err != nil || result.Status != Answered || e.prepared != 0 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestLongConversationAnswer(t *testing.T) {
	answer := strings.Repeat("Detailed answer. ", 2048) + "End."
	g := &generator{replies: []string{answer}}
	result, err := fixture(g, &executor{}, &journal{}).Run(context.Background(), "Explain in detail", approve, emit)
	if err != nil || result.Status != Answered || result.Text != answer {
		t.Fatalf("long answer bytes=%d status=%s: %v", len(result.Text), result.Status, err)
	}
}

func TestNativeArgumentsHonorConfiguredBudget(t *testing.T) {
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": strings.Repeat("x", 2000)}), "Need a smaller change", "Need a smaller change"}}
	e := &executor{}
	r := fixture(g, e, &journal{})
	r.Limits = DefaultLimits()
	r.Limits.MaxArgumentBytes = 1024
	result, err := r.Run(context.Background(), "create a", approve, emit)
	if err != nil || e.prepared != 0 || result.Status != Answered || !result.HadToolFailure {
		t.Fatalf("prepared=%d result=%+v: %v", e.prepared, result, err)
	}
}
