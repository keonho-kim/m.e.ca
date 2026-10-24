// Purpose: Verify OpenRouter metadata, authenticated chat streaming and reasoning controls.
// Pattern: Local HTTP contract tests.
// Usage: go test ./internal/provider/httpclient -run TestOpenRouter.
// Related: internal/provider/discovery/openrouter.go, internal/provider/protocol/request.go.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestOpenRouterModelMetadata(t *testing.T) {
	c := mockClient(t, connection.OpenRouter, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/models" || r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Errorf("unexpected metadata request: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"data":[{"id":"publisher/model","canonical_slug":"publisher/model-202610","name":"Publisher: Model","context_length":131072,"top_provider":{"context_length":65536,"max_completion_tokens":4096},"reasoning":{"supported_efforts":["high","low"],"default_effort":"low","mandatory":false}}]}`)
	})
	models, err := catalogFor(c).ListModels(context.Background())
	if err != nil || len(models) != 1 || models[0].ContextWindow != 65536 || models[0].MaxOutput != 4096 {
		t.Fatalf("models=%+v err=%v", models, err)
	}
	info, err := prepareModel(c, context.Background(), "publisher/model")
	if err != nil || info.Reasoning == nil || !info.Reasoning.On || !info.Reasoning.Off || info.Reasoning.Default != "low" || strings.Join(info.Reasoning.Levels, ",") != "high,low" {
		t.Fatalf("metadata=%+v err=%v", info, err)
	}
	report, err := catalogFor(c).Identify(context.Background(), "publisher/model")
	if err != nil || strings.Join(report.Originals(), ",") != "publisher/model-202610" {
		t.Fatalf("identity=%+v err=%v", report, err)
	}
}

func TestOpenRouterChatStreaming(t *testing.T) {
	on := true
	for _, test := range []struct {
		name     string
		thinking *bool
	}{{"default", nil}, {"on", &on}, {"off", new(bool)}} {
		t.Run(test.name, func(t *testing.T) {
			explicit := test.thinking
			c := mockClient(t, connection.OpenRouter, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer test-secret" {
					t.Errorf("unexpected chat request: %s", r.URL.Path)
				}
				var body struct {
					Model     string
					Stream    bool
					MaxTokens int `json:"max_tokens"`
					Tools     []json.RawMessage
					Reasoning *struct {
						Enabled *bool
						Effort  string
					}
					Template json.RawMessage `json:"chat_template_kwargs"`
					Effort   json.RawMessage `json:"reasoning_effort"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body.Model != "publisher/model" || !body.Stream || body.MaxTokens != 100 || len(body.Tools) != 1 || body.Template != nil || body.Effort != nil {
					t.Errorf("invalid OpenRouter request: %+v", body)
				}
				if explicit == nil && body.Reasoning != nil || explicit != nil && (body.Reasoning == nil || body.Reasoning.Enabled == nil || *body.Reasoning.Enabled != *explicit) {
					t.Errorf("incorrect reasoning control: %+v", body.Reasoning)
				}
				if explicit != nil && *explicit && body.Reasoning != nil && body.Reasoning.Effort != "high" {
					t.Error("reasoning effort missing")
				}
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, ": OPENROUTER PROCESSING\n\n")
				fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"reasoning\":\"brief\",\"content\":null}}]}\n\n")
				fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"inspect_fixture\",\"arguments\":\"{}\"}}]},\"finish_reason\":\"tool_calls\"}]}\n\n")
				fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"tool_calls\"}],\"usage\":{\"prompt_tokens\":42,\"completion_tokens\":7}}\n\ndata: [DONE]\n\n")
			})
			if err := c.BindModel(connection.ModelInfo{ID: "publisher/model", MaxOutput: 100, Reasoning: &connection.Reasoning{On: true, Off: true, Levels: []string{"high"}}}); err != nil {
				t.Fatal(err)
			}
			req := modelapi.Request{ModelID: "publisher/model", MaxOutput: 200, Thinking: explicit, Messages: []modelapi.Message{{Role: "user", Content: "inspect"}}, Tools: []toolapi.Definition{{Name: "inspect_fixture", Description: "Inspect fixture"}}}
			if explicit != nil && *explicit {
				req.ReasoningEffort = "high"
			}
			var done modelapi.Event
			var reasoning string
			err := c.Generate(context.Background(), req, func(event modelapi.Event) error {
				reasoning += event.Thinking
				if event.Done {
					done = event
				}
				return nil
			})
			if err != nil || reasoning != "brief" || !done.Done || len(done.Calls) != 1 || done.Usage == nil || done.Usage.Input != 42 || done.Usage.Output != 7 {
				t.Fatalf("done=%+v thinking=%q err=%v", done, reasoning, err)
			}
		})
	}
}

func TestOpenRouterStreamError(t *testing.T) {
	c := mockClient(t, connection.OpenRouter, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"error\":{\"code\":402,\"message\":\"Insufficient credits\"}}\n\ndata: [DONE]\n\n")
	})
	err := c.Generate(context.Background(), modelapi.Request{ModelID: "publisher/model", MaxOutput: 100, Messages: []modelapi.Message{{Role: "user", Content: "hello"}}}, func(modelapi.Event) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "Insufficient credits") {
		t.Fatalf("stream error lost: %v", err)
	}
}
