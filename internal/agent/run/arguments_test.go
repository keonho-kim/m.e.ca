// Purpose: Verify staged argument corrections never execute wrappers or lose the current tool phase.
// Pattern: Task contract tests.
// Usage: go test ./internal/agent/run -run 'TestArgument|TestGreeting'.
// Related: internal/agent/run/arguments.go, internal/agent/run/context.go, internal/agent/prompt/argument.go.
package run

import (
	"context"
	"strings"
	"testing"
)

func TestArgumentCorrectionsKeepLiteralPhase(t *testing.T) {
	g := &generator{replies: []string{"TOOL read_file", `{"path":"file.txt"}`, `"file.txt"`, "file.txt", "CHECK read_file", "file.txt", "DONE\nInspected"}}
	e, j := &executor{}, &journal{}
	r := fixture(g, e, j)
	result, err := r.Run(context.Background(), nil, "Read file.txt", false, approve, emit)
	if err != nil || result.Status != Verified || e.executed != 2 {
		t.Fatalf("literal correction: %+v %v", result, err)
	}
	for _, index := range []int{1, 2, 3} {
		req := g.requests[index]
		if !strings.Contains(req.Messages[0].Content, "ARGUMENT") || strings.Contains(req.Messages[1].Content, "Available tools:") {
			t.Fatal("argument reply was prompted like a new tool decision")
		}
	}
	if !strings.Contains(g.requests[2].Messages[1].Content, "Previous rejected reply") {
		t.Fatal("correction omitted the rejected response")
	}
}
func TestArgumentFailureNeverPreparesATool(t *testing.T) {
	g := &generator{replies: []string{"TOOL read_file", `{"path":"file"}`, `{"path":"file"}`, `{"path":"file"}`}}
	e, j := &executor{}, &journal{}
	r := fixture(g, e, j)
	result, err := r.Run(context.Background(), nil, "Read file", false, approve, emit)
	if err == nil || e.prepared != 0 || e.executed != 0 || result.Status != Stopped || len(g.requests) != 4 {
		t.Fatalf("invalid arguments escaped the boundary: %+v %v", result, err)
	}
}
func TestGreetingDoesNotRequireATool(t *testing.T) {
	g, e, j := &generator{replies: []string{"DONE\nHello"}}, &executor{}, &journal{}
	r := fixture(g, e, j)
	result, err := r.Run(context.Background(), nil, "Hello", false, approve, emit)
	if err != nil || result.Status != Answered || e.prepared != 0 || !strings.Contains(g.requests[0].Messages[1].Content, "greeting") {
		t.Fatalf("greeting forced a tool: %+v %v", result, err)
	}
}
