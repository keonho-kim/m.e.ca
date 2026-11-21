// Purpose: Verify native calls are assembled only after completion and schemas carry declared arguments.
// Pattern: Provider protocol contract tests.
// Usage: go test ./internal/provider/protocol -run TestNative.
// Related: internal/provider/protocol/chunk.go, internal/provider/protocol/request.go.
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
	toolcatalog "github.com/keonho-kim/meca/internal/tools/local/catalog"
)

func TestNativeChatCallsAssembleAfterCompletion(t *testing.T) {
	body := "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"write_file\",\"arguments\":\"{\\\"path\\\":\\\"README.md\\\",\"}}]}}]}\n\n" +
		"data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"\\\"content\\\":\\\"# Project\\\\n\\\"}\"}}]},\"finish_reason\":\"tool_calls\"}]}\n\ndata: [DONE]\n\n"
	response := &http.Response{Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(body))}
	var calls []toolapi.Call
	err := ReadResponse(context.Background(), response, connection.VLLM, func(event modelapi.Event) error {
		if len(event.Calls) > 0 && !event.Done {
			t.Fatal("partial call escaped the stream")
		}
		calls = append(calls, event.Calls...)
		return nil
	})
	if err != nil || len(calls) != 1 || calls[0].ID != "call_1" || !json.Valid(calls[0].Arguments) {
		t.Fatalf("calls=%+v err=%v", calls, err)
	}
}

func TestNativeChatToolFinishAtEOF(t *testing.T) {
	for _, tc := range []struct {
		name, delta string
		wantCall    bool
	}{
		{
			name:     "complete call",
			delta:    "{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"function\":{\"name\":\"inspect_fixture\",\"arguments\":\"{\\\"path\\\":\\\"a\\\"}\"}}]}",
			wantCall: true,
		},
		{name: "missing call", delta: "{}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := "data: {\"choices\":[{\"delta\":" + tc.delta + ",\"finish_reason\":\"tool_calls\"}]}\n\n"
			response := &http.Response{Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(body))}
			var final modelapi.Event
			err := ReadResponse(t.Context(), response, connection.VLLM, func(event modelapi.Event) error {
				if event.Done {
					final = event
				}
				return nil
			})
			if tc.wantCall {
				if err != nil || len(final.Calls) != 1 || final.Calls[0].ID != "call_1" {
					t.Fatalf("terminal native call lost: %+v %v", final.Calls, err)
				}
				return
			}
			var invalid *modelapi.ToolCallError
			if !errors.As(err, &invalid) || final.Done {
				t.Fatalf("missing tool call was accepted: %+v %v", final, err)
			}
		})
	}
}

func TestNativeToolsAreEncoded(t *testing.T) {
	request := modelapi.Request{ModelID: "m", Tools: []toolapi.Definition{{Name: "inspect_fixture", Description: "Read a file", Parameters: []toolapi.Parameter{{Name: "path", Description: "File path", Kind: toolapi.LiteralPath, Required: true, MaxBytes: 4096}}}}}
	_, body, err := Encode(connection.VLLM, "http://server/v1", nil, request, nil, Controls{})
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Tools []struct {
			Function struct {
				Name       string
				Parameters map[string]any
			}
		}
	}
	if err := json.Unmarshal(body, &payload); err != nil || len(payload.Tools) != 1 || payload.Tools[0].Function.Name != "inspect_fixture" || payload.Tools[0].Function.Parameters["additionalProperties"] != false {
		t.Fatalf("body=%s err=%v", body, err)
	}
}

func TestActionReasonIsFirstInWireSchema(t *testing.T) {
	definitions := toolcatalog.Definitions("edit_file", "write_file", "execute_command")
	request := modelapi.Request{ModelID: "m", Tools: definitions}
	for _, kind := range []connection.ProviderKind{connection.VLLM, connection.LMStudio, connection.Claude, connection.Gemini} {
		t.Run(string(kind), func(t *testing.T) {
			_, body, err := Encode(kind, "http://server/v1", nil, request, nil, Controls{})
			if err != nil {
				t.Fatal(err)
			}
			for _, field := range []string{`"properties":{"reason":{`, `"required":["reason",`} {
				if count := strings.Count(string(body), field); count != 3 {
					t.Fatalf("%s appeared %d times, want 3", field, count)
				}
			}
		})
	}
}
