// Purpose: Verify first-party native streaming, authentication and signed tool-result continuation.
// Pattern: Channel-controlled HTTP integration tests without real providers or credentials.
// Usage: go test -race ./internal/provider/httpclient -run TestCloudProvider.
// Related: internal/provider/protocol/claude_request.go, internal/provider/protocol/gemini_request.go, internal/provider/protocol/openai_history.go.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestCloudProviderStreamingAndSignedToolContinuation(t *testing.T) {
	for _, kind := range []connection.ProviderKind{connection.OpenAI, connection.Claude, connection.Gemini} {
		t.Run(string(kind), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			observed := make(chan struct{})
			var once sync.Once
			var requests atomic.Int32
			marker := "signed-" + string(kind)
			client := mockClient(t, kind, func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
					return
				}
				if strings.Contains(string(body), "test-secret") || r.URL.Query().Has("key") {
					t.Error("credential leaked into request body or query")
				}
				path, header, value := "/v1/responses", "Authorization", "Bearer test-secret"
				if kind == connection.Claude {
					path, header, value = "/v1/messages", "x-api-key", "test-secret"
					if r.Header.Get("anthropic-version") != "2026-10-21" {
						t.Error("missing Claude API version")
					}
				}
				if kind == connection.Gemini {
					path, header, value = "/v1beta/models/alias:streamGenerateContent", "x-goog-api-key", "test-secret"
					if r.URL.Query().Get("alt") != "sse" {
						t.Error("Gemini SSE not requested")
					}
				}
				if r.URL.Path != path || r.Header.Get(header) != value {
					t.Errorf("wrong cloud request %s %s", r.URL.Path, header)
				}
				if header != "Authorization" && r.Header.Get("Authorization") != "" {
					t.Error("extra bearer credential sent")
				}
				var wire map[string]json.RawMessage
				if err := json.Unmarshal(body, &wire); err != nil {
					t.Error(err)
					return
				}
				if requests.Add(1) == 1 {
					if len(wire["tools"]) == 0 {
						t.Error("native tools omitted")
					}
					w.Header().Set("Content-Type", "text/event-stream")
					first, rest := cloudToolStream(kind)
					_, _ = io.WriteString(w, first)
					w.(http.Flusher).Flush()
					select {
					case <-observed:
					case <-r.Context().Done():
						return
					}
					_, _ = io.WriteString(w, rest)
					return
				}
				if !strings.Contains(string(body), marker) || !strings.Contains(string(body), "file output") {
					t.Error("signed continuation or tool result missing")
				}
				if kind == connection.OpenAI && (!strings.Contains(string(body), `"function_call_output"`) || wire["messages"] != nil) {
					t.Error("wrong Responses history")
				}
				if kind == connection.Claude && !strings.Contains(string(body), `"tool_result"`) {
					t.Error("missing Claude tool result")
				}
				if kind == connection.Gemini && !strings.Contains(string(body), `"functionResponse"`) {
					t.Error("missing Gemini tool result")
				}
				w.Header().Set("Content-Type", "application/json")
				writeProviderSuccess(w, kind, "answer")
			})
			request := modelapi.Request{ModelID: "alias", MaxOutput: 128, Messages: []modelapi.Message{{Role: "system", Content: "policy"}, {Role: "user", Content: "read source"}}, Tools: []toolapi.Definition{{Name: "read_file", Description: "Read a file", Parameters: []toolapi.Parameter{{Name: "path", Kind: "string", Required: true}}}}}
			var final modelapi.Event
			err := client.Generate(ctx, request, func(event modelapi.Event) error {
				if event.Thinking == "internal" {
					once.Do(func() { close(observed) })
				}
				if len(event.Calls) > 0 && !event.Done {
					t.Error("partial tool calls became executable")
				}
				if event.Done {
					final = event
				}
				return nil
			})
			if err != nil || len(final.Calls) != 1 || final.Calls[0].Name != "read_file" || len(final.ReasoningState) == 0 || final.Usage == nil || final.Usage.Input != 4 || final.Usage.Output != 2 {
				t.Fatal("first cloud generation failed", final, err)
			}
			call := final.Calls[0]
			request.Messages = append(request.Messages, modelapi.Message{Role: "assistant", Calls: final.Calls, ReasoningState: final.ReasoningState, ProviderID: client.connection.ID, ModelID: "alias"}, modelapi.Message{Role: "tool", ToolCallID: call.ID, ToolName: call.Name, Content: "file output"})
			var answer strings.Builder
			if err := client.Generate(ctx, request, func(event modelapi.Event) error { answer.WriteString(event.Delta); return nil }); err != nil || answer.String() != "answer" || requests.Load() != 2 {
				t.Fatal("tool continuation failed", answer.String(), requests.Load(), err)
			}
		})
	}
}

func cloudToolStream(kind connection.ProviderKind) (string, string) {
	frame := func(value string) string { return "data: " + value + "\n\n" }
	switch kind {
	case connection.OpenAI:
		return frame(`{"type":"response.reasoning_summary_text.delta","delta":"internal"}`),
			frame(`{"type":"response.output_item.added","output_index":1,"item":{"type":"function_call","id":"fc_1","call_id":"call_1","name":"read_file","arguments":""}}`) +
				frame(`{"type":"response.function_call_arguments.delta","item_id":"fc_1","delta":"{\"path\":\"source.txt\"}"}`) +
				frame(`{"type":"response.output_item.done","output_index":1,"item":{"type":"function_call","id":"fc_1","call_id":"call_1","name":"read_file","arguments":"{\"path\":\"source.txt\"}"}}`) +
				frame(`{"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":4,"output_tokens":2},"output":[{"type":"reasoning","id":"rs_1","encrypted_content":"signed-openai","summary":[]},{"type":"function_call","id":"fc_1","call_id":"call_1","name":"read_file","arguments":"{\"path\":\"source.txt\"}"}]}}`)
	case connection.Claude:
		return frame(`{"type":"message_start","message":{"usage":{"input_tokens":4,"output_tokens":0}}}`) +
				frame(`{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`) + frame(`{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"internal"}}`),
			frame(`{"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"signed-claude"}}`) + frame(`{"type":"content_block_stop","index":0}`) +
				frame(`{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"call_1","name":"read_file","input":{}}}`) + frame(`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"path\":\"source.txt\"}"}}`) + frame(`{"type":"content_block_stop","index":1}`) +
				frame(`{"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":2}}`) + frame(`{"type":"message_stop"}`)
	default:
		return frame(`{"candidates":[{"content":{"role":"model","parts":[{"text":"internal","thought":true}]}}]}`),
			frame(`{"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"name":"read_file","args":{"path":"source.txt"}},"thoughtSignature":"signed-gemini"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":4,"candidatesTokenCount":1,"thoughtsTokenCount":1}}`)
	}
}

func TestCloudProviderTruncatedCallsDoNotComplete(t *testing.T) {
	for _, kind := range []connection.ProviderKind{connection.OpenAI, connection.Claude, connection.Gemini} {
		t.Run(string(kind), func(t *testing.T) {
			client := mockClient(t, kind, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				first, _ := cloudToolStream(kind)
				fmt.Fprint(w, first)
			})
			done := false
			err := client.Generate(t.Context(), recoveryRequest(), func(event modelapi.Event) error { done = done || event.Done || len(event.Calls) > 0; return nil })
			if err == nil || done {
				t.Fatal("truncated cloud stream released a call", done, err)
			}
		})
	}
}
