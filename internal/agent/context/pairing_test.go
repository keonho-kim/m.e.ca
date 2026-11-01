// Purpose: Verify cached native groups are rebuilt when an incomplete or changed suffix affects them.
// Pattern: Context behavior regression over append, mutation and truncation boundaries.
// Usage: Run TestProjectionNativeSuffix and TestProjectionDuplicateCallIDs.
// Related: internal/agent/context/pairing.go, internal/agent/context/projection.go.
package context

import (
	"fmt"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestProjectionNativeSuffixInvalidatesItsWholeGroup(t *testing.T) {
	var projection Projection
	input := Input{System: "system", Projection: &projection, RedactionVersion: func() int { return 0 }, Redact: func(s string) string { return s }, MaxBytes: 1 << 20}
	assistant := modelapi.Message{Role: "assistant"}
	for i := range toolapi.MaxCalls {
		assistant.Calls = append(assistant.Calls, toolapi.Call{ID: fmt.Sprint(i), Name: "execute_command", Arguments: []byte(`{"command":"pwd"}`)})
	}
	input.Messages = append(input.Messages, assistant)
	for i := range toolapi.MaxCalls - 1 {
		input.Messages = append(input.Messages, modelapi.Message{Role: "tool", ToolCallID: fmt.Sprint(i), ToolName: "execute_command", Content: "result"})
	}
	first, err := Build(input)
	if err != nil || first.Messages[1].Role != "user" {
		t.Fatal("incomplete group was not quoted", err)
	}
	input.Messages = append(input.Messages, modelapi.Message{Role: "tool", ToolCallID: fmt.Sprint(toolapi.MaxCalls - 1), ToolName: "execute_command", Content: "last result"})
	complete, err := Build(input)
	if err != nil || complete.Messages[1].Role != "assistant" || complete.Messages[2].Role != "tool" {
		t.Fatal("appended result did not complete native group", err)
	}
	input.Messages[toolapi.MaxCalls/2].ToolCallID = "wrong_call"
	changed, err := Build(input)
	if err != nil || changed.Messages[1].Role != "user" || changed.Messages[2].Role != "user" {
		t.Fatal("changed result left a stale native prefix", err)
	}
	if complete.Messages[1].Role != "assistant" || complete.Messages[2].Role != "tool" {
		t.Fatal("a later request mutated an earlier request's headers")
	}
	input.Messages = input.Messages[:1]
	truncated, err := Build(input)
	if err != nil || len(truncated.Messages) != 2 || truncated.Messages[1].Role != "user" {
		t.Fatal("truncated result group was replayed", err)
	}
}

func TestProjectionDuplicateCallIDsRemainQuoted(t *testing.T) {
	call := toolapi.Call{ID: "same", Name: "execute_command", Arguments: []byte(`{"command":"pwd"}`)}
	input := Input{MaxBytes: 8192, Redact: func(s string) string { return s }, Messages: []modelapi.Message{
		{Role: "assistant", Calls: []toolapi.Call{call, call}},
		{Role: "tool", ToolCallID: "same", ToolName: call.Name},
		{Role: "tool", ToolCallID: "same", ToolName: call.Name},
	}}
	request, err := Build(input)
	if err != nil {
		t.Fatal(err)
	}
	for _, message := range request.Messages[1:] {
		if message.Role != "user" {
			t.Fatal("duplicate native identities escaped as a valid group")
		}
	}
}
