// Purpose: Verify native calls are assembled only after completion and schemas carry declared arguments.
// Pattern: Provider protocol contract tests.
// Usage: go test ./internal/provider/protocol -run TestNative.
// Related: internal/provider/protocol/chunk.go, internal/provider/protocol/request.go.
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
