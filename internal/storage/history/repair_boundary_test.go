// Purpose: Keep the source observation required to correct the latest failed edit across compaction.
// Pattern: Reverse-tail selection contract test.
// Usage: go test ./internal/storage/history -run TestTailKeepsRepair.
// Related: internal/storage/history/boundary.go.
package history

import (
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"testing"
)

func TestTailKeepsRepairReadAndNativePairs(t *testing.T) {
	s := TailSelection{Target: 1}
	if _, done := s.Keep(4, 100, modelapi.Message{Role: "tool", ToolName: "edit_file", ToolCallID: "edit", Content: `{"Failure":{"code":"text_not_found"}}`}); done {
		t.Fatal("cut before failed call")
	}
	if _, done := s.Keep(3, 100, modelapi.Message{Role: "assistant", Calls: []tool.Call{{ID: "edit", Name: "edit_file", Arguments: []byte(`{"path":"a"}`)}}}); done {
		t.Fatal("repair source was compacted")
	}
	if _, done := s.Keep(2, 100, modelapi.Message{Role: "tool", ToolName: "execute_command", ToolCallID: "read"}); done {
		t.Fatal("cut before read call")
	}
	if boundary, done := s.Keep(1, 100, modelapi.Message{Role: "assistant", Calls: []tool.Call{{ID: "read", Name: "execute_command", Arguments: []byte(`{"command":"cat a"}`)}}}); !done || boundary != 0 {
		t.Fatalf("boundary %d done %v", boundary, done)
	}
}

func TestTailKeepsRepairSourcePastLaterDraft(t *testing.T) {
	s := TailSelection{Target: 150}
	steps := []modelapi.Message{
		{Role: "assistant", Content: "A draft after the failed change"},
		{Role: "tool", ToolName: "edit_file", ToolCallID: "edit", Content: `{"Failure":{"code":"text_not_found"}}`},
		{Role: "assistant", Calls: []tool.Call{{ID: "edit", Name: "edit_file", Arguments: []byte(`{"path":"a"}`)}}},
		{Role: "tool", ToolName: "execute_command", ToolCallID: "read"},
		{Role: "assistant", Calls: []tool.Call{{ID: "read", Name: "execute_command", Arguments: []byte(`{"command":"cat a"}`)}}},
	}
	for i, message := range steps {
		boundary, done := s.Keep(int64(5-i), 100, message)
		if i < 4 && done {
			t.Fatal("discarded the source needed by the latest edit failure")
		}
		if i == 4 && (!done || boundary != 0) {
			t.Fatalf("boundary=%d done=%v", boundary, done)
		}
	}
}
