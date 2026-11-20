// Purpose: Verify native call/result formats and completion boundaries for Ollama and Responses.
// Pattern: Provider wire contract tests.
// Usage: go test ./internal/provider/protocol -run TestNative.
// Related: internal/provider/protocol/tools.go, internal/provider/protocol/responses/calls.go.
package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

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
	var capacity *modelapi.ContextLimitError
	if !errors.As(err, &capacity) {
		t.Fatalf("capacity signal lost: %v", err)
	}
}

func TestAuthoritativeResponseCannotReleaseUnconfirmedCall(t *testing.T) {
	for _, provider := range []connection.ProviderKind{connection.OpenAI, connection.LMStudio} {
		for _, output := range []string{`[]`, `[{"type":"message","content":[]}]`} {
			s := streamState{provider: provider}
			emit := func(e modelapi.Event) error {
				if len(e.Calls) > 0 {
					t.Fatal("orphan tool call released")
				}
				return nil
			}
			add := `{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","id":"item","call_id":"call","name":"write_file","arguments":"{}"}}`
			if err := s.consume([]byte(add), false, emit); err != nil {
				t.Fatal(err)
			}
			last := `{"type":"response.completed","response":{"status":"completed","output":` + output + `}}`
			var malformed *modelapi.ToolCallError
			if err := s.consume([]byte(last), false, emit); !errors.As(err, &malformed) || s.Finished {
				t.Fatalf("%s output=%s err=%v", provider, output, err)
			}
		}
	}
}

func TestTruncationCanReportPartialTextAndToolOnlyAtCompletion(t *testing.T) {
	for _, tool := range []string{"", `,{"type":"function_call","arguments":"{"}`} {
		s := streamState{provider: connection.OpenAI}
		last := `{"type":"response.incomplete","response":{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[{"type":"message","content":[{"type":"output_text","text":"Partial"}]}` + tool + `]}}`
		text := ""
		err := s.consume([]byte(last), false, func(e modelapi.Event) error {
			if e.Done || len(e.Calls) > 0 {
				t.Fatal("released incomplete output as completed")
			}
			text += e.Delta
			return nil
		})
		var limit *modelapi.OutputLimitError
		if !errors.As(err, &limit) || !limit.TokenLimit || limit.PartialTool != (tool != "") || text != "Partial" {
			t.Fatalf("err=%v limit=%+v text=%q", err, limit, text)
		}
	}
}

func TestOllamaDecodeThroughputUsesEvalDuration(t *testing.T) {
	body := `{"message":{"content":"답"},"done":true,"prompt_eval_count":200,"eval_count":40,"eval_duration":500000000,"total_duration":4500000000}`
	response := &http.Response{Header: http.Header{"Content-Type": {"application/x-ndjson"}}, Body: io.NopCloser(strings.NewReader(body))}
	var completed modelapi.Event
	if err := ReadResponse(t.Context(), response, connection.Ollama, func(e modelapi.Event) error {
		if e.Done {
			completed = e
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if completed.Usage == nil || completed.Usage.Output != 40 || completed.DecodeDuration != 500*time.Millisecond {
		t.Fatalf("completion=%+v", completed)
	}
}

func TestFunctionArgumentsCannotChangeOutputIndex(t *testing.T) {
	s := streamState{provider: connection.OpenAI}
	ignore := func(modelapi.Event) error { return nil }
	add := `{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","id":"item","call_id":"call","name":"write_file"}}`
	if err := s.consume([]byte(add), false, ignore); err != nil {
		t.Fatal(err)
	}
	argument := `{"type":"response.function_call_arguments.delta","item_id":"item","output_index":1,"delta":"{}"}`
	var malformed *modelapi.ToolCallError
	if err := s.consume([]byte(argument), false, ignore); !errors.As(err, &malformed) {
		t.Fatalf("mismatched call index: %v", err)
	}
}
