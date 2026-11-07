// Purpose: Verify first-party response text, usage, tool identities and native history order.
// Pattern: In-memory provider contract regression tests.
// Usage: go test ./internal/provider/protocol -run TestCloud.
// Related: internal/provider/protocol/gemini_request.go, internal/provider/protocol/responses/stream.go.
package protocol

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func cloudRead(t *testing.T, kind connection.ProviderKind, sse bool, frames ...string) ([]modelapi.Event, error) {
	t.Helper()
	body := strings.Join(frames, "")
	header := http.Header{}
	if sse {
		header.Set("Content-Type", "text/event-stream")
		body = "data: " + strings.Join(frames, "\n\ndata: ") + "\n\n"
	}
	var events []modelapi.Event
	err := ReadResponse(context.Background(), &http.Response{Header: header, Body: io.NopCloser(strings.NewReader(body))}, kind, func(e modelapi.Event) error { events = append(events, e); return nil })
	return events, err
}

func TestCloudGeminiToolResultKeepsName(t *testing.T) {
	messages, err := Messages(connection.Gemini, []modelapi.Message{
		{Role: "assistant", Calls: []toolapi.Call{{ID: "call", Name: "read_file", Arguments: json.RawMessage(`{"path":"a"}`)}}},
		{Role: "tool", ToolCallID: "call", ToolName: "read_file", Content: "contents"},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, body, err := Encode(connection.Gemini, "https://example.com/v1beta", nil, modelapi.Request{ModelID: "alias", MaxOutput: 32}, messages, Controls{})
	if err != nil || !strings.Contains(string(body), `"name":"read_file","response"`) {
		t.Fatalf("invalid function response: %s, %v", body, err)
	}
}

func TestCloudGeminiNoArgumentCallsComplete(t *testing.T) {
	for _, args := range []string{"", `,"args":null`, `,"args":{}`} {
		events, err := cloudRead(t, connection.Gemini, true, `{"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"name":"list_command_runs"`+args+`},"thoughtSignature":"signed"}]},"finishReason":"STOP"}]}`)
		if err != nil || len(events) == 0 {
			t.Fatalf("zero-argument call: %v", err)
		}
		last := events[len(events)-1]
		if !last.Done || len(last.Calls) != 1 || string(last.Calls[0].Arguments) != "{}" {
			t.Fatalf("incomplete call: %+v", last)
		}
		messages, err := Messages(connection.Gemini, []modelapi.Message{{Role: "assistant", Calls: last.Calls, ReasoningState: last.ReasoningState}})
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := Encode(connection.Gemini, "https://example.com/v1beta", nil, modelapi.Request{ModelID: "alias", MaxOutput: 32}, messages, Controls{}); err != nil {
			t.Fatalf("signed empty arguments could not replay: %v", err)
		}
	}
}

func TestCloudOpenAIRefusalAndCompleteSummary(t *testing.T) {
	final := `{"status":"completed","output":[{"type":"reasoning","summary":[{"type":"summary_text","text":"summary"}],"encrypted_content":"cipher"},{"type":"message","content":[{"type":"refusal","refusal":"I cannot comply."}]}]}`
	for _, sse := range []bool{false, true} {
		frames := []string{final}
		if sse {
			frames = []string{`{"type":"response.reasoning_summary_text.delta","delta":"summary"}`, `{"type":"response.refusal.delta","delta":"I cannot comply."}`, `{"type":"response.completed","response":` + final + `}`}
		}
		events, err := cloudRead(t, connection.OpenAI, sse, frames...)
		var text, thinking strings.Builder
		for _, e := range events {
			text.WriteString(e.Delta)
			thinking.WriteString(e.Thinking)
		}
		if err != nil || text.String() != "I cannot comply." || thinking.String() != "summary" {
			t.Errorf("sse=%v text=%q thinking=%q err=%v", sse, text.String(), thinking.String(), err)
		}
	}
}

func TestCloudGeminiThoughtOnlyUsage(t *testing.T) {
	events, err := cloudRead(t, connection.Gemini, true, `{"candidates":[{"content":{"role":"model","parts":[{"text":"thought","thought":true}]},"finishReason":"MAX_TOKENS"}],"usageMetadata":{"promptTokenCount":4,"thoughtsTokenCount":5,"totalTokenCount":9}}`)
	if err == nil {
		t.Fatal("token limit was ignored")
	}
	for _, e := range events {
		if e.Usage != nil && e.Usage.Input == 4 && e.Usage.Output == 5 {
			return
		}
	}
	t.Fatal("reported thought-only usage was lost")
}

func TestCloudOpenAIHistoryKeepsMessageOrder(t *testing.T) {
	m := Message{Role: "assistant", Content: "beforeafter", continuation: json.RawMessage(`[{"type":"message","content":[{"type":"output_text","text":"before"}]},{"type":"reasoning","id":"rs","encrypted_content":"cipher","summary":[]},{"type":"message","content":[{"type":"output_text","text":"after"}]}]`)}
	items, err := openAIHistory(m)
	if err != nil || len(items) != 3 {
		t.Fatalf("response boundaries changed: %+v %v", items, err)
	}
	encoded, err := json.Marshal(items)
	if err != nil || strings.Contains(string(encoded), "beforeafter") {
		t.Fatal("message content was coalesced", err)
	}
}
