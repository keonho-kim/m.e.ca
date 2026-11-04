// Purpose: Preserve session closure and instruction rejection as distinct recorded outcomes.
// Pattern: Scripted runtime lifecycle regressions.
// Usage: Runtime tests inspect paired replies and actual execution records.
// Related: internal/agent/run/batch_feedback.go, internal/agent/run/execution_result.go.
package run

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	model "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/tool"
)

func TestClosedSessionDoesNotSuggestRetryingSkippedSiblings(t *testing.T) {
	tools := &executor{outcomes: []tool.Result{{Depart: true, Content: json.RawMessage("{}")}}}
	j := &journal{}
	r := fixture(&generator{}, tools, j)
	r.Generator = boundaryModel(func(_ context.Context, _ model.Request, emit func(model.Event) error) error {
		return emit(model.Event{Done: true, Calls: []tool.Call{
			batchCall("close", "inspect_fixture", map[string]any{"path": "contribution"}),
			batchCall("after", "write_file", map[string]any{"path": "after", "content": "obsolete"}),
		}})
	})
	result, err := r.Run(context.Background(), "Finish the contribution", approve, emit)
	if err != nil || !result.Departed || tools.executed != 1 || tools.prepared != 1 {
		t.Fatal("closed session continued executing", result, err)
	}
	for _, message := range j.messages {
		if message.Role != "tool" || message.ToolCallID != "after" {
			continue
		}
		var body struct{ Failure *tool.Failure }
		if err := json.Unmarshal([]byte(message.Content), &body); err != nil {
			t.Fatal(err)
		}
		if body.Failure == nil || body.Failure.Code != "session_closed" || !strings.Contains(body.Failure.Next, "Do not retry") {
			t.Fatal("closure was described as repairable tool failure", message.Content)
		}
		return
	}
	t.Fatal("closed sibling did not receive its native result")
}

func TestInstructionGuardRecordsNoFileMutation(t *testing.T) {
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "file", "content": "obsolete"}), "Preserved existing files"}}
	tools := &executor{outcomes: []tool.Result{{Failed: true, Failure: tool.NewInstructionChanged()}}}
	j := &journal{}
	r := fixture(g, tools, j)
	if _, err := r.Run(context.Background(), "Update file", approve, emit); err != nil {
		t.Fatal(err)
	}
	for _, event := range j.events {
		if op := event.Operation; op != nil && op.Failure != nil && op.Failure.Code == tool.InstructionChanged {
			if op.Started || len(op.Changes) != 0 {
				t.Fatal("instruction rejection was recorded as a mutation", op)
			}
			return
		}
	}
	t.Fatal("instruction rejection was not preserved")
}
