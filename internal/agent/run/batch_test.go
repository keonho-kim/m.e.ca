// Purpose: Verify invalid multi-call batches receive paired failures without partial execution.
// Pattern: Runtime trust-boundary test.
// Usage: go test ./internal/agent/run -run TestNativeBatch.
// Related: internal/agent/run/tool_reply.go, internal/agent/run/action.go.
package run

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type batchGenerator struct{ requests []modelapi.Request }

func (g *batchGenerator) Generate(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	g.requests = append(g.requests, request)
	if len(g.requests) > 1 {
		return emit(modelapi.Event{Delta: "Correct the invalid operation", Done: true})
	}
	return emit(modelapi.Event{Done: true, Calls: []toolapi.Call{
		{ID: "a", Name: "write_file", Arguments: json.RawMessage(`{"path":"a","content":"first"}`)},
		{ID: "b", Name: "write_file", Arguments: json.RawMessage(`{"path":"b"}`)},
	}})
}
func TestNativeBatchRejectsInvalidArgumentsBeforeAnyPreparation(t *testing.T) {
	g, e, j := &batchGenerator{}, &executor{}, &journal{}
	r := fixture(&generator{}, e, j)
	r.Generator = g
	result, err := r.Run(context.Background(), "create two files", approve, emit)
	if err != nil || e.prepared != 0 || e.executed != 0 || result.Status != runrecord.Answered || !result.HadToolFailure || len(g.requests) != 2 {
		t.Fatalf("result=%+v prepared=%d executed=%d: %v", result, e.prepared, e.executed, err)
	}
	results := map[string]string{}
	for _, message := range g.requests[1].Messages {
		if message.Role == "tool" && strings.Contains(message.Content, `"executed":false`) {
			results[message.ToolCallID] = message.Content
		}
	}
	if len(results) != 2 || !strings.Contains(results["a"], "another call in this batch") ||
		!strings.Contains(results["b"], `"field":"content"`) || !strings.Contains(results["b"], "missing content argument") {
		t.Fatalf("misattributed paired rejection: %v", results)
	}
}

func TestRejectedBatchCommitsFeedbackWithItsAssistantCall(t *testing.T) {
	g, e, j := &batchGenerator{}, &executor{}, &journal{failKind: "tool_failed"}
	r := fixture(&generator{}, e, j)
	r.Generator = g
	if _, err := r.Run(t.Context(), "Create two files.", approve, emit); err == nil || e.prepared != 0 || len(g.requests) != 1 {
		t.Fatalf("failed rejection save did not stop before preparation: requests=%d prepared=%d err=%v", len(g.requests), e.prepared, err)
	}
	for _, message := range j.messages {
		if message.Role == "tool" || len(message.Calls) > 0 {
			t.Fatalf("partial rejected call group committed: %+v", message)
		}
	}
}
