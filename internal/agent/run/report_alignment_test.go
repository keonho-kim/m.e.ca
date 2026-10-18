// Purpose: Verify a stopped run reports runtime facts without asking the model to redefine success.
// Pattern: Scripted terminal-state regression test.
// Usage: go test ./internal/agent/run -run TestStoppedReport.
// Related: internal/agent/run/report.go, internal/agent/run/finish_response.go.
package run

import (
	"context"
	"strings"
	"testing"
)

func TestStoppedReportCannotBecomeModelSuccess(t *testing.T) {
	g := &generator{replies: []string{"", "", "", "Everything succeeded <tool_call><function=write_file>"}}
	result, err := fixture(g, &executor{}, &journal{}).Run(context.Background(), "inspect a", approve, emit)
	if err != nil || result.Status != Stopped || len(g.requests) != 3 || strings.Contains(result.Text, "Everything succeeded") || !strings.Contains(result.Text, "progress") {
		t.Fatalf("%+v calls=%d %v", result, len(g.requests), err)
	}
}

func TestPseudoToolTextGetsOneCorrectionAndIsNeverFinal(t *testing.T) {
	pseudo := "I will now edit. <tool_call><function=edit_file><parameter=path>a</parameter></function></tool_call>"
	g := &generator{replies: []string{nativeCall("inspect_fixture", map[string]any{"path": "a"}), pseudo, pseudo}}
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
	example := `{"tool":"inspect_fixture","arguments":{"path":"example.txt"}}`
	g := &generator{replies: []string{example}}
	e := &executor{}
	result, err := fixture(g, e, &journal{}).Run(context.Background(), "Show an example tool request, do not execute it", approve, emit)
	if err != nil || result.Text != example || e.executed != 0 || len(g.requests) != 1 {
		t.Fatalf("%+v %v", result, err)
	}
}
