// Purpose: Verify LiteLLM Proxy discovery, bearer authentication, aliases and streamed reasoning.
// Pattern: HTTP contract tests.
// Usage: go test ./internal/provider/httpclient -run TestLiteLLM.
// Related: internal/provider/connection/connection.go, internal/provider/protocol/request.go, internal/provider/protocol/stream.go.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"net/http"
	"testing"
)

func TestLiteLLMProxyDialogueAndReasoning(t *testing.T) {
	client := mockClient(t, modelconn.LiteLLM, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Error("proxy bearer key missing")
		}
		if r.Method == http.MethodGet && r.URL.Path == "/v1/models" {
			fmt.Fprint(w, `{"data":[{"id":"team-coder"}]}`)
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected route: %s %s", r.Method, r.URL.Path)
			return
		}
		var body struct {
			Model    string             `json:"model"`
			Stream   bool               `json:"stream"`
			Messages []modelapi.Message `json:"messages"`
			Tools    json.RawMessage    `json:"tools"`
			Template json.RawMessage    `json:"chat_template_kwargs"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if body.Model != "team-coder" || !body.Stream || len(body.Tools) > 0 || len(body.Template) > 0 {
			t.Errorf("incorrect proxy request: %+v", body)
		}
		text := "Hello!"
		w.Header().Set("Content-Type", "text/event-stream")
		encoded, err := json.Marshal(map[string]any{"choices": []any{map[string]any{"delta": map[string]string{"content": text, "reasoning_content": "inspect"}}}})
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", encoded)
	})
	models, err := catalogFor(client).ListModels(context.Background())
	if err != nil || len(models) != 1 || models[0].ID != "team-coder" {
		t.Fatalf("models: %+v %v", models, err)
	}
	caps, err := diagnose(client, context.Background(), "team-coder")
	if err != nil || !caps.Streaming {
		t.Fatalf("diagnostics: %+v %v", caps, err)
	}
	var thinking string
	err = client.Generate(context.Background(), modelapi.Request{ModelID: "team-coder", MaxOutput: 128, Messages: []modelapi.Message{{Role: "user", Content: "hello"}}}, func(event modelapi.Event) error { thinking += event.Thinking; return nil })
	if err != nil || thinking != "inspect" {
		t.Fatalf("reasoning: %q %v", thinking, err)
	}
	supported, err := thinkingControl(client, context.Background(), "team-coder")
	if err != nil || supported {
		t.Fatal("unknown routed model received guessed Thinking control")
	}
}
