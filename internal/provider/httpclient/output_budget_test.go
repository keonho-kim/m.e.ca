// Purpose: Verify automatic output is resolved before each provider wire encoding.
// Pattern: HTTP contract tests.
// Usage: go test ./internal/provider/httpclient -run TestAutomaticOutput.
// Related: internal/provider/httpclient/context.go, internal/agent/model/output.go.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/provider/connection"
)

func TestAutomaticOutput(t *testing.T) {
	for _, kind := range []connection.ProviderKind{connection.Ollama, connection.LMStudio, connection.VLLM, connection.LiteLLM} {
		t.Run(string(kind), func(t *testing.T) {
			c := mockClient(t, kind, func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					MaxTokens int `json:"max_tokens"`
					MaxOutput int `json:"max_output_tokens"`
					Options   struct {
						Predict int `json:"num_predict"`
					} `json:"options"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					return
				}
				got := body.MaxTokens
				if kind == connection.Ollama {
					got = body.Options.Predict
				}
				if kind == connection.LMStudio {
					got = body.MaxOutput
				}
				if got != 196608 {
					t.Errorf("output=%d, want 196608", got)
				}
				switch kind {
				case connection.Ollama:
					fmt.Fprint(w, `{"message":{"content":"answer"},"done":true}`)
				case connection.LMStudio:
					fmt.Fprint(w, `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"answer"}]}]}`)
				default:
					fmt.Fprint(w, `{"choices":[{"message":{"content":"answer"},"finish_reason":"stop"}]}`)
				}
			})
			if err := c.Generate(context.Background(), modelapi.Request{ModelID: "alias", Messages: []modelapi.Message{{Role: "user", Content: "hello"}}}, func(modelapi.Event) error { return nil }); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAutomaticOutputUsesTokenizerCapacity(t *testing.T) {
	c := mockClient(t, connection.VLLM, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tokenize" {
			fmt.Fprint(w, `{"count":100,"max_model_len":4096}`)
			return
		}
		var body struct {
			MaxTokens int `json:"max_tokens"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if body.MaxTokens != 3072 {
			t.Errorf("output=%d, want 3072", body.MaxTokens)
		}
		fmt.Fprint(w, `{"choices":[{"message":{"content":"answer"},"finish_reason":"stop"}]}`)
	})
	if err := c.BindModel(connection.ModelInfo{ID: "alias", ContextWindow: 8192}); err != nil {
		t.Fatal(err)
	}
	if err := c.Generate(context.Background(), modelapi.Request{ModelID: "alias", Messages: []modelapi.Message{{Role: "user", Content: "hello"}}}, func(modelapi.Event) error { return nil }); err != nil {
		t.Fatal(err)
	}
}
