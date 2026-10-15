// Purpose: Verify a stopped run reports runtime facts without asking the model to redefine success.
// Pattern: Scripted terminal-state regression test.
// Usage: go test ./internal/agent/run -run TestStoppedReport.
// Related: internal/agent/run/report.go, internal/agent/run/completion_review.go.
package run

import (
	"context"
	"errors"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"strings"
	"testing"
)

func TestStoppedReportCannotBecomeModelSuccess(t *testing.T) {
	g := &generator{replies: []string{nativeCall("read_file", map[string]any{"path": "a"}), "Done", "Everything succeeded <tool_call><function=write_file>"}}
	g.completion = func(context.Context, modelapi.Request, func(modelapi.Event) error) error {
		return errors.New("review unavailable")
	}
	result, err := fixture(g, &executor{}, &journal{}).Run(context.Background(), "inspect a", approve, emit)
	if err != nil || result.Status != Stopped || len(g.requests) != 2 || strings.Contains(result.Text, "Everything succeeded") || !strings.Contains(result.Text, "review unavailable") {
		t.Fatalf("%+v calls=%d %v", result, len(g.requests), err)
	}
}

func TestPseudoToolTextGetsOneCorrectionAndIsNeverFinal(t *testing.T) {
	pseudo := "I will now edit. <tool_call><function=edit_file><parameter=path>a</parameter></function></tool_call>"
	g := &generator{replies: []string{nativeCall("read_file", map[string]any{"path": "a"}), pseudo, pseudo}}
	e, j := &executor{}, &journal{}
	result, err := fixture(g, e, j).Run(context.Background(), "inspect a", approve, emit)
	if err != nil || result.Status != Stopped || e.executed != 1 || strings.Contains(result.Text, "<tool_call>") {
		t.Fatalf("%+v %v", result, err)
	}
	for _, m := range j.messages {
		if m.Role == "assistant" && strings.Contains(m.Content, "<tool_call>") {
			t.Fatal("text tool syntax was committed as a final response")
		}
	}
}

func TestToolJSONExampleWithoutWorkIsOrdinaryConversation(t *testing.T) {
	example := `{"tool":"read_file","arguments":{"path":"example.txt"}}`
	g := &generator{replies: []string{example}}
	e := &executor{}
	result, err := fixture(g, e, &journal{}).Run(context.Background(), "Show an example tool request, do not execute it", approve, emit)
	if err != nil || result.Text != example || e.executed != 0 || len(g.requests) != 1 {
		t.Fatalf("%+v %v", result, err)
	}
}
