// Purpose: Verify repeated reads reference only bodies still present in the active context.
// Pattern: Context projection regression tests.
// Usage: go test ./internal/agent/run -run TestReadReference.
// Related: internal/agent/run/read_reference.go.
package run

import (
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"testing"
)

func TestReadReferenceRequiresActiveOriginalBody(t *testing.T) {
	body := `{"path":"a","content":"same"}`
	s := execution{definitions: (&executor{}).Definitions(), messages: []modelapi.Message{{Role: "tool", ToolName: "read_file", ToolCallID: "original", Content: body}}}
	call := toolapi.Call{Name: "read_file"}
	if got := s.contextResult(call, body, true); !strings.Contains(got, `"source_call_id":"original"`) {
		t.Fatalf("reference missing: %s", got)
	}
	s.messages = nil
	if got := s.contextResult(call, body, true); got != body {
		t.Fatalf("body unavailable after compaction: %s", got)
	}
	s.messages = []modelapi.Message{{Role: "tool", ToolName: "read_file", ToolCallID: "old", Content: body}}
	if got := s.contextResult(call, body, false); got != body {
		t.Fatal("failed read incorrectly reused")
	}
}

func TestReadReferenceReturnsBodyAfterFailure(t *testing.T) {
	s := execution{definitions: (&executor{}).Definitions(), messages: []modelapi.Message{
		{Role: "tool", ToolName: "read_file", ToolCallID: "first", Content: "source"},
		{Role: "tool", ToolName: "edit_file", ToolCallID: "failed", Content: `{"Failure":{"code":"anchor_unavailable","detail":"mismatched"}}`},
	}}
	if got := s.contextResult(toolapi.Call{Name: "read_file"}, "source", true); got != "source" {
		t.Fatalf("repair lost anchors: %s", got)
	}
}
