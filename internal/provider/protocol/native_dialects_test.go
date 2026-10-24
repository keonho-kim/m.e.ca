// Purpose: Verify native call/result formats and completion boundaries for Ollama and Responses.
// Pattern: Provider wire contract tests.
// Usage: go test ./internal/provider/protocol -run TestNative.
// Related: internal/provider/protocol/tools.go, internal/provider/protocol/lmstudio_calls.go.
package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestNativeProviderCallsAndResults(t *testing.T) {
	for _, test := range []struct {
		kind connection.ProviderKind
		body string
	}{
		{connection.Ollama, `{"message":{"tool_calls":[{"function":{"name":"inspect_fixture","arguments":{"path":"a"}}}]},"done":true}`},
		{connection.LMStudio, `{"status":"completed","output":[{"type":"function_call","id":"item_1","call_id":"call_1","name":"inspect_fixture","arguments":"{\"path\":\"a\"}"}]}`},
	} {
		t.Run(string(test.kind), func(t *testing.T) {
			var calls []toolapi.Call
			response := &http.Response{Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(test.body))}
			if err := ReadResponse(context.Background(), response, test.kind, func(e modelapi.Event) error { calls = append(calls, e.Calls...); return nil }); err != nil || len(calls) != 1 {
				t.Fatalf("calls=%+v err=%v", calls, err)
			}
			messages, err := Messages(test.kind, []modelapi.Message{{Role: "assistant", Calls: calls}, {Role: "tool", ToolCallID: calls[0].ID, ToolName: "inspect_fixture", Content: "result"}})
			if err != nil {
				t.Fatal(err)
			}
			_, body, err := Encode(test.kind, "http://server", nil, modelapi.Request{ModelID: "m", MaxOutput: 128}, messages, Controls{})
			if err != nil {
				t.Fatal(err)
			}
			if test.kind == connection.LMStudio {
				if !strings.Contains(string(body), `"type":"function_call_output"`) || !strings.Contains(string(body), `"call_id":"call_1"`) {
					t.Fatal(string(body))
				}
			} else {
				if !strings.Contains(string(body), `"arguments":{"path":"a"}`) || !strings.Contains(string(body), `"tool_name":"inspect_fixture"`) {
					t.Fatal(string(body))
				}
			}
		})
	}
}

func TestNativeResponsesStreamDoesNotDuplicateCalls(t *testing.T) {
	frames := []string{
		`{"type":"response.output_item.added","output_index":1,"item":{"type":"function_call","id":"fc_1","call_id":"call_1","name":"inspect_fixture","arguments":""}}`,
		`{"type":"response.function_call_arguments.delta","item_id":"fc_1","delta":"{\"path\":"}`,
		`{"type":"response.function_call_arguments.delta","item_id":"fc_1","delta":"\"a\"}"}`,
		`{"type":"response.function_call_arguments.done","item_id":"fc_1","arguments":"{\"path\":\"a\"}"}`,
		`{"type":"response.output_item.done","output_index":1,"item":{"type":"function_call","id":"fc_1","call_id":"call_1","name":"inspect_fixture","arguments":"{\"path\":\"a\"}"}}`,
		`{"type":"response.completed","response":{"status":"completed","output":[{"type":"reasoning"},{"type":"function_call","id":"fc_1","call_id":"call_1","name":"inspect_fixture","arguments":"{\"path\":\"a\"}"}]}}`,
	}
	var body strings.Builder
	for _, frame := range frames {
		body.WriteString("data: " + frame + "\n\n")
	}
	response := &http.Response{Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(body.String()))}
	var calls []toolapi.Call
	err := ReadResponse(context.Background(), response, connection.LMStudio, func(e modelapi.Event) error { calls = append(calls, e.Calls...); return nil })
	if err != nil || len(calls) != 1 || string(calls[0].Arguments) != `{"path":"a"}` {
		t.Fatalf("calls=%+v err=%v", calls, err)
	}
}

func TestNativeTruncatedArgumentsNeverEmitCall(t *testing.T) {
	body := `{"choices":[{"message":{"tool_calls":[{"id":"call_1","type":"function","function":{"name":"write_file","arguments":"{\"path\":"}}]},"finish_reason":"length"}]}`
	response := &http.Response{Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
	count := 0
	err := ReadResponse(context.Background(), response, connection.VLLM, func(e modelapi.Event) error { count += len(e.Calls); return nil })
	if err == nil || count != 0 {
		t.Fatalf("calls=%d err=%v", count, err)
	}
}

func TestNativeSchemaAndMessagesPreserveLiteralArguments(t *testing.T) {
	call := toolapi.Call{ID: "c", Name: "write_file", Arguments: json.RawMessage(`{"path":"a","content":"\n  keep\n"}`)}
	messages, err := Messages(connection.VLLM, []modelapi.Message{{Role: "assistant", Calls: []toolapi.Call{call}}, {Role: "tool", ToolCallID: "c", ToolName: "write_file", Content: "ok"}})
	if err != nil || messages[1].ToolCallID != "c" {
		t.Fatalf("%+v %v", messages, err)
	}
	var calls []struct{ Function struct{ Arguments string } }
	if err := json.Unmarshal(messages[0].Calls, &calls); err != nil || calls[0].Function.Arguments != string(call.Arguments) {
		t.Fatalf("%+v %v", calls, err)
	}
}

func TestNativeContextLimitSignalSurvivesResponseFailure(t *testing.T) {
	body := `data: {"type":"response.failed","response":{"status":"failed","error":{"code":"context_length_exceeded","message":"capacity"}}}` + "\n\n"
	response := &http.Response{Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(body))}
	err := ReadResponse(context.Background(), response, connection.LMStudio, func(modelapi.Event) error { return nil })
	var capacity interface{ ContextLimit() bool }
	if !errors.As(err, &capacity) || !capacity.ContextLimit() {
		t.Fatalf("capacity signal lost: %v", err)
	}
}
